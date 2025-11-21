package sender

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gobot/core/logger"
	"gobot/core/telegram/netutil"

	tele "gopkg.in/telebot.v4"
)

var (
	// ErrQueueClosed is returned when enqueue is attempted after dispatcher stop.
	ErrQueueClosed = errors.New("telegram sender: queue closed")
	// ErrQueueFull indicates the queue is saturated and the job was not accepted.
	ErrQueueFull = errors.New("telegram sender: queue full")

	tokenRe = regexp.MustCompile(`bot[0-9]+:[A-Za-z0-9_-]+`)
)

// Options controls the behaviour of the outbound dispatcher.
type Options struct {
	QueueSize    int
	Workers      int
	MaxRetries   int
	RetryBackoff time.Duration
	// MaxDuration bounds the time spent retrying a single job.
	MaxDuration time.Duration
}

type job struct {
	ctx      context.Context
	action   string
	endpoint string
	run      func() error
}

// Dispatcher executes outbound Telegram calls asynchronously with retries.
type Dispatcher struct {
	opts Options
	jobs chan job
	stop chan struct{}
	once sync.Once
	wg   sync.WaitGroup
	errs atomic.Uint64
}

// NewDispatcher starts a dispatcher with sane defaults if options are zeroed.
func NewDispatcher(opts Options) *Dispatcher {
	if opts.QueueSize <= 0 {
		opts.QueueSize = 256
	}
	if opts.Workers <= 0 {
		opts.Workers = 4
	}
	if opts.MaxRetries < 0 {
		opts.MaxRetries = 0
	}
	if opts.RetryBackoff <= 0 {
		opts.RetryBackoff = 2 * time.Second
	}
	if opts.MaxDuration <= 0 {
		opts.MaxDuration = 12 * time.Second
	}

	d := &Dispatcher{
		opts: opts,
		jobs: make(chan job, opts.QueueSize),
		stop: make(chan struct{}),
	}

	d.wg.Add(opts.Workers)
	for i := 0; i < opts.Workers; i++ {
		go d.worker()
	}

	return d
}

// Enqueue schedules the provided function for asynchronous execution.
// The run closure must be idempotent if retries are desired.
func (d *Dispatcher) Enqueue(ctx context.Context, action, endpoint string, run func() error) error {
	if run == nil {
		return errors.New("telegram sender: nil run function")
	}
	select {
	case <-d.stop:
		return ErrQueueClosed
	default:
	}

	j := job{
		ctx:      ctx,
		action:   action,
		endpoint: endpoint,
		run:      run,
	}

	select {
	case d.jobs <- j:
		return nil
	default:
		return ErrQueueFull
	}
}

// ErrorCount returns the number of failed jobs.
func (d *Dispatcher) ErrorCount() uint64 {
	return d.errs.Load()
}

// Close stops workers and waits for them to finish processing queued jobs.
func (d *Dispatcher) Close() {
	d.once.Do(func() {
		close(d.stop)
		close(d.jobs)
		d.wg.Wait()
	})
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for j := range d.jobs {
		d.handleJob(j)
	}
}

func (d *Dispatcher) handleJob(j job) {
	ctx := j.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, d.opts.MaxDuration)
	defer cancel()

	start := time.Now()
	logger.Debug(ctx, "tg.sender", "send.start", sendLogAttrs(ctx, j)...)

	var (
		lastErr       error
		failureLogged bool
	)
	attempts := d.opts.MaxRetries + 1

attemptLoop:
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := deadlineCtx.Err(); err != nil {
			lastErr = err
			break
		}

		if err := j.run(); err != nil {
			lastErr = err
			if !netutil.ShouldRetry(err) || attempt == attempts {
				logSendFailure(ctx, j, lastErr, attempts, time.Since(start))
				failureLogged = true
				break
			}

			delay := d.opts.RetryBackoff * time.Duration(attempt)
			timer := time.NewTimer(delay)
			select {
			case <-deadlineCtx.Done():
				timer.Stop()
				lastErr = deadlineCtx.Err()
				logSendFailure(ctx, j, lastErr, attempts, time.Since(start))
				failureLogged = true
				break attemptLoop
			case <-timer.C:
			}
			logger.Debug(ctx, "tg.sender", "send.retry.backoff",
				append(sendLogAttrs(ctx, j),
					slog.Int("attempt", attempt),
					slog.Duration("delay", delay),
				)...,
			)
			continue
		}

		// Success
		if attempt > 1 {
			logger.Info(ctx, "tg.sender", "send.retry.success",
				append(sendLogAttrs(ctx, j),
					slog.Int("attempt", attempt),
					slog.Int("elapsed_ms", durationToMS(time.Since(start))),
				)...,
			)
		}
		logSendSuccess(ctx, j, attempt, time.Since(start))
		return
	}

	if lastErr != nil {
		d.errs.Add(1)
		if !failureLogged {
			logSendFailure(ctx, j, lastErr, attempts, time.Since(start))
		}
	}
}

func sendLogAttrs(ctx context.Context, j job) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("action", j.action),
	}
	if j.endpoint != "" {
		attrs = append(attrs, slog.String("endpoint", j.endpoint))
	}
	if rid := logger.RIDFrom(ctx); rid != "" {
		attrs = append(attrs, slog.String("rid", rid))
	}
	if updateID := logger.UpdateIDFrom(ctx); updateID != 0 {
		attrs = append(attrs, slog.Int("update_id", updateID))
	}
	if chatID := logger.ChatIDFrom(ctx); chatID != 0 {
		attrs = append(attrs, slog.Int64("chat_id", chatID))
	}
	if userID := logger.UserIDFrom(ctx); userID != 0 {
		attrs = append(attrs, slog.Int64("user_id", userID))
	}
	return attrs
}

func logSendSuccess(ctx context.Context, j job, attempt int, elapsed time.Duration) {
	attrs := sendLogAttrs(ctx, j)
	if attempt > 1 {
		attrs = append(attrs, slog.Int("attempt", attempt))
	}
	attrs = append(attrs, slog.Int("elapsed_ms", durationToMS(elapsed)))
	logger.Debug(ctx, "tg.sender", "send.success", attrs...)
}

func logSendFailure(ctx context.Context, j job, err error, attempts int, elapsed time.Duration) {
	attrs := sendLogAttrs(ctx, j)
	attrs = append(attrs,
		slog.String("error", sanitizeErrorMessage(err)),
		slog.String("error_kind", classifyError(err)),
		slog.Int("elapsed_ms", durationToMS(elapsed)),
	)
	if attempts > 0 {
		attrs = append(attrs, slog.Int("attempts", attempts))
	}
	logger.Error(ctx, "tg.sender", "send.fail", attrs...)
}

func durationToMS(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(logger.RoundMS(d) / time.Millisecond)
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsTimeout {
			return "timeout"
		}
		return "dns"
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Timeout() {
			return "timeout"
		}
		if opErr.Op == "dial" {
			return "dial"
		}
		if opErr.Op == "read" || opErr.Op == "write" {
			if kind := classifyError(opErr.Err); kind != "" && kind != "unknown" {
				return kind
			}
		}
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return "timeout"
		}
		if urlErr.Err != nil && !errors.Is(urlErr.Err, err) {
			if kind := classifyError(urlErr.Err); kind != "" && kind != "unknown" {
				return kind
			}
		}
	}

	var alertErr tls.AlertError
	if errors.As(err, &alertErr) {
		return "tls"
	}

	status := httpStatusFromError(err)
	switch {
	case status >= 500:
		return "http_5xx"
	case status >= 400:
		return "http_4xx"
	}

	return "unknown"
}

// sanitizeErrorMessage prevents accidental leakage of Telegram bot tokens in logs.
func sanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if msg == "" {
		return ""
	}
	return tokenRe.ReplaceAllString(msg, "bot<redacted>")
}

func httpStatusFromError(err error) int {
	if err == nil {
		return 0
	}

	var apiErr *tele.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}

	var floodErr tele.FloodError
	if errors.As(err, &floodErr) {
		return http.StatusTooManyRequests
	}

	var groupErr tele.GroupError
	if errors.As(err, &groupErr) {
		return http.StatusBadRequest
	}

	msg := err.Error()
	if msg == "" {
		return 0
	}

	lastOpen := strings.LastIndex(msg, "(")
	lastClose := strings.LastIndex(msg, ")")
	if lastOpen >= 0 && lastClose > lastOpen+1 {
		codeStr := strings.TrimSpace(msg[lastOpen+1 : lastClose])
		if code, convErr := strconv.Atoi(codeStr); convErr == nil {
			return code
		}
	}

	return 0
}
