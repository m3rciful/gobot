package logger

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode"
)

// contextKey is a private type to avoid collisions in context.
type contextKey string

const (
	ctxRID      contextKey = "rid"
	ctxUpdateID contextKey = "update_id"
	ctxUserID   contextKey = "user_id"
	ctxChatID   contextKey = "chat_id"
	ctxLogger   contextKey = "logger"
	ctxHandler  contextKey = "handler"
	ctxTraceID  contextKey = "trace_id"
	ctxSpanID   contextKey = "span_id"
)

// WithLogger stores the provided slog.Logger in context for propagation across layers.
func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if log == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxLogger, log)
}

// FromContext extracts slog.Logger from context or returns global default.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return L
	}
	if v := ctx.Value(ctxLogger); v != nil {
		if l, ok := v.(*slog.Logger); ok {
			return l
		}
	}
	return L
}

// WithRID attaches request correlation id into context.
func WithRID(ctx context.Context, rid string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxRID, rid)
}

// RIDFrom extracts rid from context if present.
func RIDFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(ctxRID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithUpdateMeta attaches common update identifiers to context.
func WithUpdateMeta(ctx context.Context, updateID int, userID, chatID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, ctxUpdateID, updateID)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxChatID, chatID)
	return ctx
}

// WithHandler stores handler identifier in context for downstream logs.
func WithHandler(ctx context.Context, handler string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if handler == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxHandler, handler)
}

// HandlerFrom returns handler identifier from context if present.
func HandlerFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(ctxHandler); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// WithTrace attaches trace and span identifiers to context.
func WithTrace(ctx context.Context, traceID, spanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID != "" {
		ctx = context.WithValue(ctx, ctxTraceID, traceID)
	}
	if spanID != "" {
		ctx = context.WithValue(ctx, ctxSpanID, spanID)
	}
	return ctx
}

// TraceIDFrom extracts trace id from context.
func TraceIDFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(ctxTraceID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SpanIDFrom extracts span id from context.
func SpanIDFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(ctxSpanID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// UserIDFrom extracts Telegram user ID from context.
func UserIDFrom(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if v := ctx.Value(ctxUserID); v != nil {
		switch id := v.(type) {
		case int64:
			return id
		case int:
			return int64(id)
		}
	}
	return 0
}

// ChatIDFrom extracts chat id from context.
func ChatIDFrom(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if v := ctx.Value(ctxChatID); v != nil {
		switch id := v.(type) {
		case int64:
			return id
		case int:
			return int64(id)
		}
	}
	return 0
}

// UpdateIDFrom extracts update identifier from context.
func UpdateIDFrom(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	if v := ctx.Value(ctxUpdateID); v != nil {
		switch id := v.(type) {
		case int:
			return id
		case int64:
			return int(id)
		}
	}
	return 0
}

// Sanitize trims non-printable runes from s to keep logs clean.
// It removes control characters (Unicode categories Cc, Cf) except for tab and newline.
func Sanitize(s string) string {
	if s == "" {
		return s
	}
	b := strings.Builder{}
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			// skip
			continue
		}
		// also skip DEL character
		if r == 0x7F {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// SanitizeLimit applies Sanitize and limits the output length in runes.
func SanitizeLimit(s string, max int) string {
	if max <= 0 {
		return ""
	}
	cleaned := Sanitize(s)
	// fast path
	if len([]rune(cleaned)) <= max {
		return cleaned
	}
	r := []rune(cleaned)
	return string(r[:max])
}

// BuildRID returns a correlation identifier in the format updateID:chatID:userID.
func BuildRID(updateID int, chatID, userID int64) string {
	return fmt.Sprintf("%d:%d:%d", updateID, chatID, userID)
}

// CompactRID shortens colon-separated RID into base36 segments for readability.
// When the input does not match the expected format it is returned unchanged.
func CompactRID(rid string) string {
	rid = strings.TrimSpace(rid)
	if rid == "" {
		return ""
	}
	parts := strings.Split(rid, ":")
	if len(parts) != 3 {
		return rid
	}
	compact := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return rid
		}
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return rid
		}
		compact = append(compact, strings.ToLower(strconv.FormatInt(n, 36)))
	}
	return strings.Join(compact, ".")
}
