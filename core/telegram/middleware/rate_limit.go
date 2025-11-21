package middleware

import (
	"sync"
	"time"

	"gobot/core/logger"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// RateLimitOptions configures behaviour of the rate limit middleware.
type RateLimitOptions struct {
	Interval  time.Duration
	Exclude   map[string]struct{}
	OnLimited tele.HandlerFunc
}

// RateLimitMiddleware returns a middleware that enforces a minimum interval
// between messages from the same user.
func RateLimitMiddleware(opts RateLimitOptions) tele.MiddlewareFunc {
	var (
		userLastSeen   = make(map[int64]time.Time)
		userLastSeenMu sync.Mutex
	)
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			user := c.Sender()
			if user == nil || opts.Interval <= 0 {
				return next(c)
			}

			// Determine update kind and apply configured exclusions
			upd := c.Update()
			kind := "other"
			switch {
			case upd.Callback != nil:
				kind = "callback"
			case upd.Message != nil:
				kind = "message"
			case upd.Query != nil:
				kind = "inline_query"
			}
			if _, skip := opts.Exclude[kind]; skip {
				return next(c)
			}

			now := time.Now()

			userLastSeenMu.Lock()
			if last, ok := userLastSeen[user.ID]; ok && now.Sub(last) < opts.Interval {
				userLastSeenMu.Unlock()
				chat := c.Chat()
				if chat != nil {
					logger.TG.Warn("rate limit",
						slog.String("event", "tg.rate_limit"),
						slog.Int64("chat_id", chat.ID),
						slog.Int64("user_id", user.ID),
					)
				} else {
					logger.TG.Warn("rate limit",
						slog.String("event", "tg.rate_limit"),
						slog.Int64("user_id", user.ID),
					)
				}
				if opts.OnLimited != nil {
					_ = opts.OnLimited(c)
				}
				return nil
			}

			userLastSeen[user.ID] = now
			userLastSeenMu.Unlock()
			return next(c)
		}
	}
}
