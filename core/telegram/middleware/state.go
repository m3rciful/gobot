package middleware

import (
	"gobot/core/logger"
	tghelpers "gobot/core/telegram/helpers"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// StateGetter is the minimal interface required from an FSM manager.
type StateGetter interface {
	GetState(userID int64) string
}

// State returns a middleware that checks if user is in the expected FSM state.
func State(mgr StateGetter, expectedState string) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			userID := c.Sender().ID
			currentState := mgr.GetState(userID)
			ctx := tghelpers.BuildContext(c)
			if currentState == expectedState {
				logger.TG.LogAttrs(ctx, slog.LevelDebug, "fsm.match",
					slog.Int64("user_id", userID),
					slog.String("state", currentState),
					slog.String("expected", expectedState),
					slog.String("rid", logger.RIDFrom(ctx)),
				)
				return next(c)
			}
			logger.TG.LogAttrs(ctx, slog.LevelDebug, "fsm.skip",
				slog.Int64("user_id", userID),
				slog.String("state", currentState),
				slog.String("expected", expectedState),
				slog.String("rid", logger.RIDFrom(ctx)),
			)
			// Ignore message if user is in a different state
			return nil
		}
	}
}

// StateMiddleware (deprecated duplicate) removed; use State instead.
