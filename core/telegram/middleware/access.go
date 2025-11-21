package middleware

import tele "gopkg.in/telebot.v4"

// AdminOptions defines how admin-only checks should behave.
type AdminOptions struct {
	AdminID  int64
	OnReject tele.HandlerFunc
}

// WithAdminCheck wraps a command handler enforcing admin-only execution when required.
func WithAdminCheck(opts AdminOptions, cmd struct {
	AdminOnly bool
	Handler   tele.HandlerFunc
}) tele.HandlerFunc {
	if !cmd.AdminOnly || opts.AdminID == 0 {
		return cmd.Handler
	}
	return func(c tele.Context) error {
		if int64(c.Sender().ID) != opts.AdminID {
			if opts.OnReject != nil {
				return opts.OnReject(c)
			}
			return nil
		}
		return cmd.Handler(c)
	}
}

// AdminOnlyMiddleware ensures that only the admin user can invoke downstream handlers.
func AdminOnlyMiddleware(opts AdminOptions) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if opts.AdminID != 0 && int64(c.Sender().ID) != opts.AdminID {
				if opts.OnReject != nil {
					return opts.OnReject(c)
				}
				return nil
			}
			return next(c)
		}
	}
}
