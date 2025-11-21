package logger

import (
	"strings"
	"time"
)

// Status maps error to a unified status string for logs.
func Status(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

// Took returns rounded duration since start for compact logging.
func Took(start time.Time) time.Duration {
	return RoundMS(time.Since(start))
}

// RoundMS rounds duration to the nearest millisecond for consistent logging.
func RoundMS(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return d.Round(time.Millisecond)
}

// SummarizeStrings joins up to limit elements and reports whether truncation happened.
func SummarizeStrings(values []string, limit int) (string, bool) {
	if limit <= 0 {
		return "", len(values) > 0
	}
	if len(values) <= limit {
		return strings.Join(values, ", "), false
	}
	return strings.Join(values[:limit], ", "), true
}
