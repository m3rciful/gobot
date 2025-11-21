package logger

import (
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"gobot/core/buildinfo"
	coreconfig "gobot/core/config"
)

var (
	initOnce   sync.Once
	shutdownMu sync.Mutex
	shutdowned bool

	logWriter  *asyncWriter
	logClosers []io.Closer

	levelVar slog.LevelVar

	debugSampler  = newRatioSampler(1, 50)
	traceOverride bool

	// L is the base logger exposed for compatibility while migrating to context-first logging.
	L *slog.Logger

	// DB logs database-related events for legacy call sites.
	DB *slog.Logger
	// TG logs Telegram transport events for legacy call sites.
	TG *slog.Logger
	// MIG logs database migration events for legacy call sites.
	MIG *slog.Logger
	// TWire logs Telegram wiring steps for legacy call sites.
	TWire *slog.Logger
	// SEED logs database seeding operations for legacy call sites.
	SEED *slog.Logger
	// SVCUsers logs user service activity for legacy call sites.
	SVCUsers *slog.Logger
	// SVCVehicles logs vehicle service activity for legacy call sites.
	SVCVehicles *slog.Logger
	// SVCFuelTypes logs fuel type service activity for legacy call sites.
	SVCFuelTypes *slog.Logger
	// SVCVehicleFuels logs vehicle fuel service activity for legacy call sites.
	SVCVehicleFuels *slog.Logger
	// SVCRefuels logs refuel service activity for legacy call sites.
	SVCRefuels *slog.Logger
	// SVCCatalog logs service catalog activity for legacy call sites.
	SVCCatalog *slog.Logger
	// SVCServiceEvents logs service event activity for legacy call sites.
	SVCServiceEvents *slog.Logger
)

// InitLogger configures the global structured logger. It may be called only once.
func InitLogger(cfg *coreconfig.Config) error {
	var initErr error
	initOnce.Do(func() {
		format := selectFormat(cfg)
		order := selectKeyOrder(cfg)
		level := selectLevel(cfg)
		levelVar.Set(level)

		num, den := parseDebugSample(cfg)
		debugSampler.Set(num, den)
		traceOverride = detectTraceFlag()

		outputs, closers, err := buildOutputs(cfg)
		if err != nil {
			initErr = err
			return
		}
		logClosers = closers
		logWriter = newAsyncWriter(outputs, 64*1024)

		handler := newStructuredHandler(handlerConfig{
			level:    &levelVar,
			writer:   logWriter,
			format:   format,
			keyOrder: order,
		})

		logger := slog.New(handler)
		L = logger
		slog.SetDefault(logger)

		wireLegacyComponents()
		logStartup(cfg)
	})
	return initErr
}

func wireLegacyComponents() {
	if L == nil {
		return
	}
	DB = L.With("component", "db")
	TG = L.With("component", "tg")
	MIG = L.With("component", "db.migrate")
	TWire = L.With("component", "tg.wire")
	SEED = L.With("component", "db.seed")
	SVCUsers = L.With("component", "service.users")
	SVCVehicles = L.With("component", "service.vehicles")
	SVCFuelTypes = L.With("component", "service.fuel_types")
	SVCVehicleFuels = L.With("component", "service.vehicle_fuels")
	SVCRefuels = L.With("component", "service.refuels")
	SVCCatalog = L.With("component", "service.catalog")
	SVCServiceEvents = L.With("component", "service.events")
}

func logStartup(cfg *coreconfig.Config) {
	if L == nil {
		return
	}
	attrs := []slog.Attr{
		slog.String("component", "app"),
		slog.String("event", "startup"),
		slog.String("go_version", runtime.Version()),
		slog.String("build_commit", buildinfo.Commit),
		slog.String("build_time", buildinfo.Date),
	}
	if cfg != nil {
		attrs = append(attrs,
			slog.String("cfg_profile", selectProfile(cfg)),
		)
	}
	L.LogAttrs(context.Background(), slog.LevelInfo, "startup", attrs...)
}

// Shutdown flushes buffered log output and closes opened sinks.
func Shutdown() error {
	shutdownMu.Lock()
	defer shutdownMu.Unlock()
	if shutdowned {
		return nil
	}
	shutdowned = true

	var errs []error
	if logWriter != nil {
		if err := logWriter.Flush(); err != nil {
			errs = append(errs, err)
		}
		if err := logWriter.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	for _, c := range logClosers {
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func selectFormat(cfg *coreconfig.Config) logFormat {
	if cfg == nil {
		return formatJSON
	}
	raw := strings.ToLower(strings.TrimSpace(cfg.Logging.Format))
	switch raw {
	case "kv", "text", "pretty":
		return formatKV
	case "json":
		return formatJSON
	}
	// Prefer human-friendly format when profile indicates debug/dev mode.
	if strings.EqualFold(cfg.Logging.Profile, "debug") || strings.EqualFold(cfg.Logging.Profile, "dev") {
		return formatKV
	}
	return formatJSON
}

func selectKeyOrder(cfg *coreconfig.Config) []string {
	if cfg == nil {
		return append([]string(nil), defaultKeyOrder...)
	}
	raw := strings.TrimSpace(cfg.Logging.KeysOrder)
	if raw == "" || raw == "default" {
		return append([]string(nil), defaultKeyOrder...)
	}
	parts := strings.Split(raw, ",")
	order := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		order = append(order, trimmed)
	}
	if len(order) == 0 {
		return append([]string(nil), defaultKeyOrder...)
	}
	return order
}

func selectLevel(cfg *coreconfig.Config) slog.Level {
	if cfg == nil {
		return slog.LevelInfo
	}
	raw := strings.ToLower(strings.TrimSpace(cfg.Logging.Level))
	switch raw {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

func buildOutputs(cfg *coreconfig.Config) ([]io.Writer, []io.Closer, error) {
	writers := []io.Writer{os.Stdout}
	var closers []io.Closer
	if cfg == nil {
		return writers, closers, nil
	}
	dir := strings.TrimSpace(cfg.Logging.Dir)
	file := strings.TrimSpace(cfg.Logging.BotFile)
	if dir != "" && file != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("logger: failed to create log dir %s: %v", dir, err)
		} else {
			path := filepath.Join(dir, file)
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				log.Printf("logger: failed to open log file %s: %v", path, err)
			} else {
				writers = append(writers, f)
				closers = append(closers, f)
			}
		}
	}
	return writers, closers, nil
}

func selectProfile(cfg *coreconfig.Config) string {
	if cfg == nil {
		return ""
	}
	if profile := strings.TrimSpace(cfg.Logging.Profile); profile != "" {
		return strings.ToLower(profile)
	}
	return "prod"
}

// Background returns context.Background() provided for compatibility with existing call sites.
func Background() context.Context {
	return context.Background()
}

// LogEvent preserves legacy helper to ensure event attribute presence with context-aware logging.
func LogEvent(ctx context.Context, logg *slog.Logger, level slog.Level, event string, attrs ...slog.Attr) {
	if logg == nil {
		logg = FromContext(ctx)
	}
	if logg == nil {
		logg = L
	}
	if logg == nil {
		return
	}
	if event != "" {
		attrs = append([]slog.Attr{slog.String("event", event)}, attrs...)
	}
	logg.LogAttrs(ctx, level, "", attrs...)
}

// Component constructs a logger scoped to the provided component attribute.
func Component(name string) *slog.Logger {
	if L == nil {
		return nil
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return L
	}
	return L.With("component", trimmed)
}

// Event logs with component scope resolved automatically.
func Event(ctx context.Context, component string, level slog.Level, event string, attrs ...slog.Attr) {
	logg := Component(component)
	if logg == nil {
		logg = FromContext(ctx)
		if logg != nil && strings.TrimSpace(component) != "" {
			logg = logg.With("component", strings.TrimSpace(component))
		}
	}
	LogEvent(ctx, logg, level, event, attrs...)
}

// Debug logs a debug-level event for the given component.
func Debug(ctx context.Context, component, event string, attrs ...slog.Attr) {
	Event(ctx, component, slog.LevelDebug, event, attrs...)
}

// Info logs an info-level event for the given component.
func Info(ctx context.Context, component, event string, attrs ...slog.Attr) {
	Event(ctx, component, slog.LevelInfo, event, attrs...)
}

// Warn logs a warn-level event for the given component.
func Warn(ctx context.Context, component, event string, attrs ...slog.Attr) {
	Event(ctx, component, slog.LevelWarn, event, attrs...)
}

// Error logs an error-level event for the given component.
func Error(ctx context.Context, component, event string, attrs ...slog.Attr) {
	Event(ctx, component, slog.LevelError, event, attrs...)
}

func parseDebugSample(cfg *coreconfig.Config) (int, int) {
	if cfg == nil {
		return 1, 50
	}
	spec := strings.TrimSpace(cfg.Logging.DebugSample)
	if spec == "" {
		return 1, 50
	}
	num, den := parseRatioSpec(spec)
	if num == 0 && den == 0 {
		return 0, 0
	}
	if num <= 0 || den <= 0 {
		return 1, 50
	}
	return num, den
}

func detectTraceFlag() bool {
	return isTruthy(os.Getenv("TRACE")) || isTruthy(os.Getenv("LOG_TRACE"))
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// ShouldSampleDebug reports whether debug-level details should be logged for high-volume events.
func ShouldSampleDebug() bool {
	if traceOverride {
		return true
	}
	return debugSampler.Allow()
}

// TraceEnabled indicates whether trace override is forcing full debug output.
func TraceEnabled() bool {
	return traceOverride
}
