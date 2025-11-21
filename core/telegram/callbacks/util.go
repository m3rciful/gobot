package callbacks

import (
	"strings"

	tele "gopkg.in/telebot.v4"
)

// ParseCallbackData parses Telebot's \f<unique>|<payload> encoding.
// Returns unique and payload (may be empty).
func ParseCallbackData(cb *tele.Callback) (string, string) {
	if cb == nil {
		return "", ""
	}
	raw := cb.Data
	// Telebot encodes like: \f<unique>|<payload>
	raw = strings.TrimPrefix(raw, "\\f")
	// Split once: unique | payload?
	parts := strings.SplitN(raw, "|", 2)
	unique := strings.TrimSpace(parts[0])
	payload := ""
	if len(parts) == 2 {
		payload = parts[1]
	}
	return unique, payload
}

// CallbackKey returns cb.Unique if present; otherwise parses from Data.
func CallbackKey(c tele.Context) string {
	cb := c.Callback()
	if cb == nil {
		return ""
	}
	if cb.Unique != "" {
		return cb.Unique
	}
	k, _ := ParseCallbackData(cb)
	return k
}

// CallbackPayload returns payload (after '|') parsed from Data.
func CallbackPayload(c tele.Context) string {
	cb := c.Callback()
	if cb == nil {
		return ""
	}
	// prefer cb.Data since cb.Unique may be empty in generic OnCallback
	_, payload := ParseCallbackData(cb)
	return payload
}
