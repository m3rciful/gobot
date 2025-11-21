package router

import (
	"time"

	tg "gobot/core/telegram"
	"gobot/core/telegram/middleware"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// CallbackOptions customises fallback behaviour for callbacks.
type CallbackOptions struct {
	NotFound tele.HandlerFunc
}

// CallbackRoute returns a handler that routes callbacks through the registry.
func CallbackRoute(reg *tg.Registry, opts CallbackOptions) tg.Route {
	handler := func(c tele.Context) error {
		start := time.Now()
		if c.Callback() == nil {
			return nil
		}

		key, _ := parseCallback(c.Callback())
		name := "callback." + normalizeHandlerName(key)
		extras := []slog.Attr{slog.String("cb_key", key)}

		_ = c.Respond()

		cbHandler, ok := reg.GetCallback(key)
		if !ok || cbHandler == nil {
			fallback := reg.CallbackNotFound()
			if fallback == nil {
				fallback = opts.NotFound
			}
			extras = append(extras, slog.String("reason", "not_found"))
			return handleWithSummary(c, name, start, "", "", func() error {
				if fallback != nil {
					return fallback(c)
				}
				return nil
			}, extras...)
		}

		return handleWithSummary(c, name, start, "", "", func() error {
			return cbHandler(c)
		}, extras...)
	}
	return tg.Route{
		Endpoint: tele.OnCallback,
		Handler:  middleware.RecoverMiddleware(middleware.LoggerMiddleware(handler)),
	}
}
