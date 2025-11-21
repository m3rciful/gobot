package telegram

import (
	"strings"
	"time"

	coreconfig "github.com/m3rciful/gobot/core/config"
	"github.com/m3rciful/gobot/core/telegram/middleware"

	tele "gopkg.in/telebot.v4"
)

// DefaultMiddlewares builds the shared middleware chain for bots.
func DefaultMiddlewares(cfg *coreconfig.Config, onLimited func(tele.Context) error) []Middleware {
	mws := []Middleware{
		{Name: "recover", Use: middleware.RecoverMiddleware},
	}

	if cfg != nil {
		interval := time.Duration(cfg.RateLimit.IntervalMS) * time.Millisecond
		if interval > 0 {
			ex := make(map[string]struct{}, len(cfg.RateLimit.ExcludeUpdates))
			for _, t := range cfg.RateLimit.ExcludeUpdates {
				ex[strings.ToLower(t)] = struct{}{}
			}
			opts := middleware.RateLimitOptions{
				Interval: interval,
				Exclude:  ex,
			}
			if onLimited != nil {
				opts.OnLimited = onLimited
			}
			mws = append(mws, Middleware{
				Name: "rate_limit",
				Use:  middleware.RateLimitMiddleware(opts),
			})
		}
	}

	mws = append(mws,
		Middleware{Name: "logger", Use: middleware.LoggerMiddleware},
		Middleware{Name: "metrics", Use: middleware.MessageMetricsMiddleware},
	)

	return mws
}
