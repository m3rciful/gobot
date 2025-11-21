package logger

import "strings"

const (
	// LevelDebug represents the debug severity level name.
	LevelDebug = "DEBUG"
	// LevelInfo represents the info severity level name.
	LevelInfo = "INFO"
	// LevelWarn represents the warning severity level name.
	LevelWarn = "WARN"
	// LevelError represents the error severity level name.
	LevelError = "ERROR"
	// LevelFatal represents the fatal severity level name.
	LevelFatal = "FATAL"
)

var allowedLevels = map[string]string{
	"debug":   LevelDebug,
	"info":    LevelInfo,
	"warn":    LevelWarn,
	"warning": LevelWarn,
	"error":   LevelError,
	"fatal":   LevelFatal,
}

var allowedStatus = map[string]string{
	"ok":           "ok",
	"fail":         "fail",
	"skip":         "skip",
	"retry":        "retry",
	"rate_limited": "rate_limited",
	"cancelled":    "cancelled",
}

var allowedCache = map[string]string{
	"hit":     "hit",
	"miss":    "miss",
	"refresh": "refresh",
}

var allowedOutcome = map[string]string{
	"ok":           "ok",
	"fail":         "fail",
	"cancelled":    "cancelled",
	"rate_limited": "rate_limited",
}

func normalizeLevel(level string) string {
	if level == "" {
		return LevelInfo
	}
	if mapped, ok := allowedLevels[strings.ToLower(level)]; ok {
		return mapped
	}
	return strings.ToUpper(level)
}

func normalizeStatus(status string) (string, bool) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return "", false
	}
	if mapped, ok := allowedStatus[status]; ok {
		return mapped, true
	}
	return status, false
}

func normalizeCache(cache string) (string, bool) {
	cache = strings.ToLower(strings.TrimSpace(cache))
	if cache == "" {
		return "", false
	}
	val, ok := allowedCache[cache]
	return val, ok
}

func normalizeOutcome(outcome string) (string, bool) {
	outcome = strings.ToLower(strings.TrimSpace(outcome))
	if outcome == "" {
		return "", false
	}
	val, ok := allowedOutcome[outcome]
	return val, ok
}

var defaultKeyOrder = []string{
	"ts",
	"level",
	"component",
	"event",
	"status",
	"rid",
	"rid_full",
	"trace_id",
	"span_id",
	"ts_unix_nano",
	"update_id",
	"user_id",
	"chat_id",
	"chat_type",
	"handler",
	"operation",
	"op",
	"cb_key",
	"outcome",
	"duration_ms",
	"messages",
	"kb",
	"count",
	"page",
	"pages",
	"cache",
	"payload",
	"lang",
	"username",
	"mode",
	"listen",
	"public_url",
	"http_code",
	"db",
	"host",
	"port",
	"vehicle_id",
	"reminder_id",
	"reminders",
	"err",
	"err_code",
	"cause",
	"retryable",
	"attempts",
	"backoff_ms",
	"rate_limited",
	"collapsed",
	"repeats",
	"pending_count",
	"refuels_shown",
	"refuels_total",
}
