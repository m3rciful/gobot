package helpers

import (
	"strings"
	"time"
)

var flexibleDateLayouts = []string{
	"2006-01-02 15:04",
	"2006-1-2 15:04",
	"2006-01-02",
	"2006-1-2",
	"02.01.2006 15:04",
	"2.1.2006 15:04",
	"02.01.2006",
	"2.1.2006",
}

// ParseFlexibleDate tries several common date formats used in Telegram flows.
// It returns the parsed time in the local timezone and true on success.
func ParseFlexibleDate(input string) (time.Time, bool) {
	s := strings.TrimSpace(input)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range flexibleDateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ParseFlexibleDateUnix returns the Unix timestamp in seconds for the parsed date.
func ParseFlexibleDateUnix(input string) (int64, bool) {
	if t, ok := ParseFlexibleDate(input); ok {
		return t.Unix(), true
	}
	return 0, false
}
