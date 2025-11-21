package state

import tele "gopkg.in/telebot.v4"

const sessionKey = "fsm_session"

// WithSession injects a session from Manager into the handler context.
func WithSession(mgr Manager) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			userID := c.Sender().ID
			session := mgr.Get(userID)

			// Store the session in context so it can be retrieved later
			c.Set(sessionKey, session)

			return next(c)
		}
	}
}
