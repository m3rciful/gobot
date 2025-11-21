package router

import (
	"time"

	tg "gobot/core/telegram"
	"gobot/core/telegram/middleware"

	tele "gopkg.in/telebot.v4"
)

// FSM defines the minimal interface for an FSM manager.
type FSM interface {
	InProgress(userID int64) bool
	ManagerHandler(c tele.Context) error
}

// TextOptions controls fallback behaviour for text/document updates.
type TextOptions struct {
	UnknownText     tele.HandlerFunc
	UnknownDocument tele.HandlerFunc
}

// TextRoutes builds handlers for text and document routing.
// The routes perform the same logic previously wired via RegisterTextRouter.
func TextRoutes(fsmMgr FSM, reg *tg.Registry, opts TextOptions) []tg.Route {
	handler := func(c tele.Context) error {
		start := time.Now()
		text := c.Text()

		if fsmMgr != nil && fsmMgr.InProgress(c.Sender().ID) {
			return handleWithSummary(c, "fsm", start, "", "", func() error {
				return fsmMgr.ManagerHandler(c)
			})
		}

		if reg != nil {
			if key, cmd, ok := reg.LookupCommand(text); ok && cmd.Handler != nil {
				name := normalizeHandlerName(key)
				return handleWithSummary(c, name, start, "", "", func() error {
					return cmd.Handler(c)
				})
			}
		}

		if reg != nil {
			if fb := reg.TextFallback(); fb != nil {
				return handleWithSummary(c, "fallback", start, "", "", func() error {
					return fb(c)
				})
			}
		}

		if opts.UnknownText != nil {
			return handleWithSummary(c, "unknown_text", start, "", "", func() error {
				return opts.UnknownText(c)
			})
		}

		logHandlerSummary(c, "unknown_text", start, "skip", "ok", nil)
		return nil
	}

	docHandler := func(c tele.Context) error {
		start := time.Now()
		if fsmMgr != nil && fsmMgr.InProgress(c.Sender().ID) {
			return handleWithSummary(c, "fsm_document", start, "", "", func() error {
				return fsmMgr.ManagerHandler(c)
			})
		}
		if opts.UnknownDocument != nil {
			return handleWithSummary(c, "unexpected_document", start, "", "", func() error {
				return opts.UnknownDocument(c)
			})
		}
		logHandlerSummary(c, "unexpected_document", start, "skip", "ok", nil)
		return nil
	}

	return []tg.Route{
		{
			Endpoint: tele.OnText,
			Handler:  middleware.RecoverMiddleware(middleware.LoggerMiddleware(handler)),
		},
		{
			Endpoint: tele.OnDocument,
			Handler:  middleware.RecoverMiddleware(middleware.LoggerMiddleware(docHandler)),
		},
	}
}
