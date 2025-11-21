package middleware

import (
	"runtime/debug"

	"gobot/core/logger"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// RecoverMiddleware catches panics in handlers and prevents the bot from crashing
func RecoverMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		defer func() {
			if r := recover(); r != nil {
				logger.TG.Error("panic recovered",
					slog.String("event", "tg.panic"),
					slog.Any("err", r),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()
		return next(c)
	}
}
