package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	coreconfig "gobot/core/config"
	"gobot/core/logger"
	coretelegram "gobot/core/telegram"

	"log/slog"
)

// ConfigCarrier exposes access to the embedded core configuration.
type ConfigCarrier interface {
	CoreConfig() *coreconfig.Config
}

// TelegramApp is the minimal interface required to run a Telegram bot.
type TelegramApp interface {
	TelegramRunOptions() (coretelegram.RunOptions, error)
}

// Options describe how to load configuration, bootstrap the app, and run the bot.
type Options struct {
	ConfigEnvVar      string
	DefaultConfigPath string

	LoadConfig func(path string) (ConfigCarrier, error)
	Bootstrap  func(cfg ConfigCarrier) (TelegramApp, error)

	ShutdownLogger func() error
	RunTelegram    func(ctx context.Context, opts coretelegram.RunOptions) error
}

// Run loads configuration, bootstraps the Telegram app, and starts the bot runtime.
func Run(opts Options) error {
	if opts.LoadConfig == nil {
		return fmt.Errorf("cmd: LoadConfig is required")
	}
	if opts.Bootstrap == nil {
		return fmt.Errorf("cmd: Bootstrap is required")
	}

	env := opts.ConfigEnvVar
	if env == "" {
		env = "CONFIG_PATH"
	}
	cfgPath := os.Getenv(env)
	if cfgPath == "" {
		cfgPath = opts.DefaultConfigPath
	}
	if cfgPath == "" {
		return fmt.Errorf("cmd: config path not provided via %s or DefaultConfigPath", env)
	}

	log.Printf("loading config: %s", cfgPath)
	cfg, err := opts.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("cmd: failed to load config: %w", err)
	}
	if cfg.CoreConfig() == nil {
		return fmt.Errorf("cmd: loaded config is missing core configuration")
	}

	application, err := opts.Bootstrap(cfg)
	if err != nil {
		return fmt.Errorf("cmd: bootstrap failed: %w", err)
	}

	shutdownLogger := opts.ShutdownLogger
	if shutdownLogger == nil {
		shutdownLogger = logger.Shutdown
	}
	defer func() {
		if err := shutdownLogger(); err != nil {
			log.Printf("logger shutdown error: %v", err)
		}
	}()

	runOpts, err := application.TelegramRunOptions()
	if err != nil {
		return fmt.Errorf("cmd: telegram options build failed: %w", err)
	}

	startedAt := time.Now()
	prevStart := runOpts.OnStart
	runOpts.OnStart = func(ctx context.Context, rt coretelegram.Runtime) error {
		if prevStart != nil {
			if err := prevStart(ctx, rt); err != nil {
				return err
			}
		}
		logger.L.With("component", "app").Info("app ready",
			slog.String("event", "ready"),
			slog.Duration("startup_duration", logger.RoundMS(time.Since(startedAt))),
		)
		return nil
	}

	prevStop := runOpts.OnStop
	runOpts.OnStop = func(ctx context.Context, rt coretelegram.Runtime) error {
		logger.L.With("component", "app").Info("shutting down...",
			slog.String("event", "shutdown"),
		)
		if prevStop != nil {
			return prevStop(ctx, rt)
		}
		return nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	run := opts.RunTelegram
	if run == nil {
		run = coretelegram.RunTelegram
	}

	return run(ctx, runOpts)
}
