package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

// TelegramConfig holds Telegram bot related settings that are common for all bots.
type TelegramConfig struct {
	Token   string `yaml:"token" envconfig:"BOT_TOKEN"`
	AdminID int64  `yaml:"admin_id" envconfig:"TELEGRAM_ADMIN_ID"`
	RunMode string `yaml:"run_mode" envconfig:"TELEGRAM_RUN_MODE"`
	// LongPollTimeoutSeconds defines long polling timeout; 0 -> default
	LongPollTimeoutSeconds int `yaml:"longpoll_timeout_seconds" envconfig:"TELEGRAM_LONGPOLL_TIMEOUT_SECONDS"`
}

// WebhookConfig specifies webhook settings.
type WebhookConfig struct {
	URL    string `yaml:"url" envconfig:"WEBHOOK_URL"`
	Listen string `yaml:"listen" envconfig:"WEBHOOK_LISTEN"`
	Port   int    `yaml:"port" envconfig:"WEBHOOK_PORT"`
}

// LoggingConfig defines logging related configuration.
type LoggingConfig struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	KeysOrder   string `yaml:"keys_order"`
	DebugSample string `yaml:"debug_sample"`
	Stacks      string `yaml:"stacks"`
	Dir         string `yaml:"dir"`
	BotFile     string `yaml:"bot_file"`
	ErrorsFile  string `yaml:"errors_file"`
	// Profile indicates environment profile such as "debug" or "prod".
	Profile string `yaml:"profile"`
}

const (
	// RunModeWebhook selects webhook mode for Telegram updates.
	RunModeWebhook = "webhook"
	// RunModeLongpoll selects long-polling mode for Telegram updates.
	RunModeLongpoll = "longpoll"
)

const (
	// UpdateCallback identifies callback updates for rate limit exclusions.
	UpdateCallback = "callback"
	// UpdateMessage identifies message updates for rate limit exclusions.
	UpdateMessage = "message"
	// UpdateInlineQuery identifies inline query updates for rate limit exclusions.
	UpdateInlineQuery = "inline_query"
)

// RateLimitConfig holds settings for rate limiting.
// ExcludeUpdates accepts update types to bypass limiting:
// - "callback": Telegram callback button presses
// - "message": standard text messages
// - "inline_query": inline query updates
type RateLimitConfig struct {
	IntervalMS     int      `yaml:"interval_ms" envconfig:"RATE_LIMIT_INTERVAL_MS"`
	ExcludeUpdates []string `yaml:"exclude_updates" envconfig:"RATE_LIMIT_EXCLUDE_UPDATES"`
}

// Config aggregates the configuration that belongs to the reusable core.
type Config struct {
	Telegram  TelegramConfig  `yaml:"telegram"`
	Webhook   WebhookConfig   `yaml:"webhook"`
	Logging   LoggingConfig   `yaml:"logging"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// Load reads configuration from a YAML file and environment variables.
func Load(path string) (*Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env: %w", err)
	}

	if err := Normalize(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Normalize performs basic validation of required configuration fields and adjusts defaults.
func Normalize(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("nil config")
	}

	if cfg.Telegram.Token == "" {
		return fmt.Errorf("telegram token is required")
	}

	rm := strings.ToLower(strings.TrimSpace(cfg.Telegram.RunMode))
	if rm == "" {
		rm = RunModeLongpoll
	}
	if rm == "polling" { // accept alias
		rm = RunModeLongpoll
	}
	switch rm {
	case RunModeWebhook:
		if strings.TrimSpace(cfg.Webhook.URL) == "" {
			return fmt.Errorf("webhook.url is required when telegram.run_mode is 'webhook'")
		}
		if strings.TrimSpace(cfg.Webhook.Listen) == "" {
			return fmt.Errorf("webhook.listen is required when telegram.run_mode is 'webhook'")
		}
		if cfg.Webhook.Port <= 0 {
			return fmt.Errorf("webhook.port must be > 0 when telegram.run_mode is 'webhook'")
		}
	case RunModeLongpoll:
		if cfg.Telegram.LongPollTimeoutSeconds < 0 {
			return fmt.Errorf("telegram.longpoll_timeout_seconds must be >= 0")
		}
	default:
		return fmt.Errorf("invalid telegram.run_mode %q; allowed: webhook, longpoll", cfg.Telegram.RunMode)
	}
	cfg.Telegram.RunMode = rm

	allowed := map[string]struct{}{
		UpdateCallback:    {},
		UpdateMessage:     {},
		UpdateInlineQuery: {},
	}
	for i, v := range cfg.RateLimit.ExcludeUpdates {
		key := strings.ToLower(strings.TrimSpace(v))
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("invalid rate_limit.exclude_updates value %q; allowed: callback, message, inline_query", v)
		}
		cfg.RateLimit.ExcludeUpdates[i] = key
	}
	return nil
}
