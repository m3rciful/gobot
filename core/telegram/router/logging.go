package router

import (
	"reflect"
	"strings"
	"time"

	"gobot/core/logger"
	tghelpers "gobot/core/telegram/helpers"
	"gobot/core/telegram/middleware"
	"log/slog"

	tele "gopkg.in/telebot.v4"
)

func handleWithSummary(c tele.Context, handlerName string, start time.Time, statusOverride, outcomeOverride string, fn func() error, extras ...slog.Attr) error {
	tghelpers.WithHandler(c, handlerName)
	err := fn()
	logHandlerSummary(c, handlerName, start, statusOverride, outcomeOverride, err, extras...)
	return err
}

func logHandlerSummary(c tele.Context, handlerName string, start time.Time, statusOverride, outcomeOverride string, err error, extras ...slog.Attr) {
	ctx := tghelpers.WithHandler(c, handlerName)
	msgs, kb := middleware.GetCounters(c)

	status := statusOverride
	if status == "" {
		if err != nil {
			status = "fail"
		} else {
			status = "ok"
		}
	}
	outcome := outcomeOverride
	if outcome == "" {
		if err != nil {
			outcome = "fail"
		} else {
			outcome = "ok"
		}
	}

	duration := logger.RoundMS(time.Since(start)).Milliseconds()
	attrs := []slog.Attr{
		slog.String("status", status),
		slog.String("handler", handlerName),
		slog.String("outcome", outcome),
		slog.Int("messages", msgs),
		slog.Bool("kb", kb),
		slog.Int64("duration_ms", duration),
	}
	if err != nil {
		attrs = append(attrs,
			slog.String("err", logger.SanitizeLimit(err.Error(), 256)),
			slog.String("err_code", deriveErrorCode(err)),
			slog.String("cause", handlerName),
		)
	}
	if len(extras) > 0 {
		attrs = append(attrs, extras...)
	}
	logger.LogEvent(ctx, logger.Component("tg"), slog.LevelInfo, "handler.handled", attrs...)
}

func normalizeHandlerName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unknown"
	}
	name = strings.TrimPrefix(name, "/")
	name = strings.ReplaceAll(name, " ", "_")
	return strings.ToLower(name)
}

func deriveErrorCode(err error) string {
	if err == nil {
		return ""
	}
	type coder interface{ Code() string }
	if c, ok := err.(coder); ok {
		code := strings.TrimSpace(c.Code())
		if code != "" {
			return strings.ToUpper(strings.ReplaceAll(code, " ", "_"))
		}
	}
	t := reflect.TypeOf(err)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t != nil {
		return strings.ToUpper(strings.ReplaceAll(t.Name(), " ", "_"))
	}
	return "UNKNOWN_ERROR"
}

func parseCallback(cb *tele.Callback) (string, string) {
	if cb == nil {
		return "", ""
	}
	if cb.Unique != "" {
		return cb.Unique, cb.Data
	}
	raw := strings.TrimPrefix(cb.Data, "\\f")
	parts := strings.SplitN(raw, "|", 2)
	key := strings.TrimSpace(parts[0])
	payload := ""
	if len(parts) == 2 {
		payload = parts[1]
	}
	return key, payload
}
