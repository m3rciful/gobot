package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	coreconfig "gobot/core/config"
	"gobot/core/logger"
	tghelpers "gobot/core/telegram/helpers"
	tgsender "gobot/core/telegram/sender"

	tele "gopkg.in/telebot.v4"
	"log/slog"
)

// Middleware describes a global bot middleware to be registered via bot.Use.
type Middleware struct {
	Name string
	Use  func(next tele.HandlerFunc) tele.HandlerFunc
}

// Route declares a single bot handler bound to an arbitrary endpoint.
// Endpoint values are passed directly to tele.Bot.Handle.
type Route struct {
	Endpoint any
	Handler  tele.HandlerFunc
}

// RunOptions controls the behaviour of RunTelegram.
type RunOptions struct {
	Config   *coreconfig.Config
	Registry *Registry

	DispatcherOptions tgsender.Options
	Dispatcher        *tgsender.Dispatcher

	Middlewares []Middleware
	Routes      []Route

	DisableWebhookCleanup bool
	DisableHelperDispatcher bool

	OnStart func(ctx context.Context, rt Runtime) error
	OnStop  func(ctx context.Context, rt Runtime) error
}

// Runtime exposes runtime components to lifecycle hooks.
type Runtime struct {
	Dispatcher *tgsender.Dispatcher
	Registry   *Registry
}

// RunTelegram composes and runs a Telegram bot until the provided context is done.
func RunTelegram(ctx context.Context, opts RunOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Config == nil {
		return fmt.Errorf("telegram: nil config provided")
	}

	cfg := opts.Config
	reg := opts.Registry
	if reg == nil {
		reg = NewRegistry()
	}

	poller := BuildPoller(PollerOptions{
		RunMode:                cfg.Telegram.RunMode,
		LongPollTimeoutSeconds: cfg.Telegram.LongPollTimeoutSeconds,
		Webhook: WebhookOptions{
			Listen: cfg.Webhook.Listen,
			Port:   cfg.Webhook.Port,
			URL:    cfg.Webhook.URL,
		},
	})

	settings := tele.Settings{
		Token:  cfg.Telegram.Token,
		Poller: poller,
		Client: BuildHTTPClient(),
	}

	buildStart := time.Now()
	bot, err := tele.NewBot(settings)
	if err != nil {
		return fmt.Errorf("telegram: bot initialization failed: %w", err)
	}
	buildTook := time.Since(buildStart)

	dispatcher := opts.Dispatcher
	if dispatcher == nil {
		dispatcher = tgsender.NewDispatcher(opts.DispatcherOptions)
	}
	useHelperDispatcher := !opts.DisableHelperDispatcher
	if useHelperDispatcher {
		tghelpers.SetDispatcher(dispatcher)
	}

	rt := Runtime{
		Dispatcher: dispatcher,
		Registry:   reg,
	}

	// Log adapter configuration (INFO aggregates only)
	switch p := poller.(type) {
	case *tele.Webhook:
		attrs := []slog.Attr{
			slog.String("event", "mode"),
			slog.String("mode", "webhook"),
			slog.String("listen", p.Listen),
			slog.String("public_url", p.Endpoint.PublicURL),
			slog.Duration("duration", logger.RoundMS(buildTook)),
		}
		logger.TG.LogAttrs(ctx, slog.LevelInfo, "webhook mode", attrs...)
	default:
		timeoutSec := 10
		if cfg.Telegram.LongPollTimeoutSeconds > 0 {
			timeoutSec = cfg.Telegram.LongPollTimeoutSeconds
		}
		logger.TG.Info("polling mode",
			slog.String("event", "mode"),
			slog.String("mode", "polling"),
			slog.Int("timeout_seconds", timeoutSec),
			slog.Duration("duration", logger.RoundMS(buildTook)),
		)

		if !opts.DisableWebhookCleanup && strings.EqualFold(cfg.Telegram.RunMode, coreconfig.RunModeLongpoll) {
			if err := deleteWebhook(cfg.Telegram.Token, false); err != nil {
				logger.TG.Warn("failed to delete webhook",
					slog.String("event", "delete_webhook"),
					slog.String("mode", "polling"),
					slog.String("err", err.Error()),
				)
			} else {
				logger.TG.Info("webhook deleted",
					slog.String("event", "delete_webhook"),
					slog.String("mode", "polling"),
				)
			}
		}
	}

	for _, mw := range opts.Middlewares {
		if mw.Use == nil {
			continue
		}
		bot.Use(mw.Use)
	}

	for _, route := range opts.Routes {
		if route.Endpoint == nil || route.Handler == nil {
			continue
		}
		bot.Handle(route.Endpoint, route.Handler)
	}

	SetupCommands(bot, reg)

	if opts.OnStart != nil {
		if err := opts.OnStart(ctx, rt); err != nil {
			dispatcher.Close()
			if useHelperDispatcher {
				tghelpers.SetDispatcher(nil)
			}
			return err
		}
	}

	runDone := make(chan struct{})
	go func() {
		bot.Start()
		close(runDone)
	}()

	var runErr error

	select {
	case <-ctx.Done():
		bot.Stop()
		<-runDone
		runErr = ctx.Err()
	case <-runDone:
	}

	var stopErr error
	if opts.OnStop != nil {
		stopCtx := ctx
		if stopCtx == nil {
			stopCtx = context.Background()
		}
		stopErr = opts.OnStop(stopCtx, rt)
	}

	dispatcher.Close()
	if useHelperDispatcher {
		tghelpers.SetDispatcher(nil)
	}

	if stopErr != nil {
		return stopErr
	}
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) {
			return nil
		}
		return runErr
	}

	return nil
}

func deleteWebhook(token string, dropPending bool) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("empty token")
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteWebhook", token)
	body := "drop_pending_updates=false"
	if dropPending {
		body = "drop_pending_updates=true"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deleteWebhook status: %s", resp.Status)
	}
	return nil
}
