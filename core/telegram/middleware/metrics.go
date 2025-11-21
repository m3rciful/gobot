package middleware

import (
	"gobot/core/logger"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

// metricsContext wraps tele.Context to count sent messages and detect keyboard usage.
type metricsContext struct{ tele.Context }

func (m metricsContext) incMessages(hasKB bool) {
	// Update messages counter
	n := 0
	if v := m.Get("messages"); v != nil {
		if nv, ok := v.(int); ok {
			n = nv
		}
	}
	m.Set("messages", n+1)
	if hasKB {
		m.Set("kb", true)
	}
}

func hasKeyboard(opts []interface{}) bool {
	for _, o := range opts {
		switch v := o.(type) {
		case *tele.SendOptions:
			if v != nil && v.ReplyMarkup != nil {
				return true
			}
		case *tele.ReplyMarkup:
			if v != nil {
				return true
			}
		}
	}
	return false
}

// Send proxies tele.Context.Send while updating message counters.
func (m metricsContext) Send(what interface{}, opts ...interface{}) error {
	err := m.Context.Send(what, opts...)
	if err == nil {
		m.incMessages(hasKeyboard(opts))
	}
	return err
}

// Reply proxies tele.Context.Reply while updating message counters.
func (m metricsContext) Reply(what interface{}, opts ...interface{}) error {
	err := m.Context.Reply(what, opts...)
	if err == nil {
		m.incMessages(hasKeyboard(opts))
	}
	return err
}

// Edit proxies tele.Context.Edit while updating message counters.
func (m metricsContext) Edit(what interface{}, opts ...interface{}) error {
	err := m.Context.Edit(what, opts...)
	if err == nil {
		// Count edits as responses as well
		m.incMessages(hasKeyboard(opts))
	}
	return err
}

// EditOrSend proxies tele.Context.EditOrSend while updating message counters.
func (m metricsContext) EditOrSend(what interface{}, opts ...interface{}) error {
	err := m.Context.EditOrSend(what, opts...)
	if err == nil {
		m.incMessages(hasKeyboard(opts))
	}
	return err
}

// EditOrReply proxies tele.Context.EditOrReply while updating message counters.
func (m metricsContext) EditOrReply(what interface{}, opts ...interface{}) error {
	err := m.Context.EditOrReply(what, opts...)
	if err == nil {
		m.incMessages(hasKeyboard(opts))
	}
	return err
}

// MessageMetricsMiddleware instruments context to track messages count and keyboard usage.
func MessageMetricsMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		// Initialize counters
		c.Set("messages", 0)
		c.Set("kb", false)
		// Wrap context
		return next(metricsContext{Context: c})
	}
}

// Optionally, log metrics warnings if needed (not used, placeholder to satisfy linter imports)
var _ = slog.Attr{}
var _ = logger.RoundMS

// GetCounters reads message count and keyboard presence flags from context.
func GetCounters(c tele.Context) (int, bool) {
	msgs := 0
	if v := c.Get("messages"); v != nil {
		if n, ok := v.(int); ok {
			msgs = n
		}
	}
	kb := false
	if v := c.Get("kb"); v != nil {
		if b, ok := v.(bool); ok {
			kb = b
		}
	}
	return msgs, kb
}
