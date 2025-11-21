package helpers

import (
	"context"

	"gobot/core/logger"

	tele "gopkg.in/telebot.v4"
)

const contextKey = "logger_ctx"

// StoreContext attaches reusable context to tele.Context for downstream helpers.
func StoreContext(c tele.Context, ctx context.Context) {
	if c == nil || ctx == nil {
		return
	}
	c.Set(contextKey, ctx)
}

// ContextFrom telegram context if previously stored by middleware.
func ContextFrom(c tele.Context) (context.Context, bool) {
	if c == nil {
		return nil, false
	}
	if v := c.Get(contextKey); v != nil {
		if ctx, ok := v.(context.Context); ok {
			return ctx, true
		}
	}
	return nil, false
}

// BuildContext constructs a context.Context from tele.Context,
// enriching it with RID and update/user/chat metadata for consistent service logging.
func BuildContext(c tele.Context) context.Context {
	if cached, ok := ContextFrom(c); ok {
		return cached
	}

	upd := c.Update()
	user := c.Sender()
	chat := c.Chat()

	var (
		chatID int64
		userID int64
	)
	if chat != nil {
		chatID = chat.ID
	}
	if user != nil {
		userID = user.ID
	}

	rid, _ := c.Get("rid").(string)
	if rid == "" {
		rid = logger.BuildRID(upd.ID, chatID, userID)
	}

	ctx := context.Background()
	ctx = logger.WithRID(ctx, rid)
	ctx = logger.WithUpdateMeta(ctx, upd.ID, userID, chatID)
	ctx = logger.WithLogger(ctx, logger.Component("tg"))
	StoreContext(c, ctx)
	return ctx
}

// WithHandler enriches stored context with handler metadata for downstream logs.
func WithHandler(c tele.Context, handler string) context.Context {
	ctx := BuildContext(c)
	if handler == "" {
		return ctx
	}
	ctx = logger.WithHandler(ctx, handler)
	StoreContext(c, ctx)
	return ctx
}
