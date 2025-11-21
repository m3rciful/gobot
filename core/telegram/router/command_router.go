package router

import (
	"github.com/m3rciful/gobot/core/logger"
	tg "github.com/m3rciful/gobot/core/telegram"
	"github.com/m3rciful/gobot/core/telegram/middleware"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// CommandRouteOptions configures how commands are wrapped and exposed.
type CommandRouteOptions struct {
	AdminID      int64
	OnAdminReject tele.HandlerFunc
}

// CommandRoutes prepares command handlers wrapped with shared middleware.
func CommandRoutes(reg *tg.Registry, opts CommandRouteOptions) []tg.Route {
	if reg == nil {
		return nil
	}

	adminOpts := middleware.AdminOptions{
		AdminID:  opts.AdminID,
		OnReject: opts.OnAdminReject,
	}

	routes := make([]tg.Route, 0, len(reg.Commands()))
	for cmd, def := range reg.Commands() {
		h := def.Handler
		h = middleware.RecoverMiddleware(h)
		h = middleware.LoggerMiddleware(h)
		if def.AdminOnly {
			h = middleware.AdminOnlyMiddleware(adminOpts)(h)
		}
		routes = append(routes, tg.Route{
			Endpoint: cmd,
			Handler:  h,
		})
	}

	logger.TWire.Info("tg.wire",
		slog.String("event", "complete"),
		slog.Int("commands", len(reg.Commands())),
		slog.Int("callbacks", len(reg.ListCallbacks())),
	)

	return routes
}
