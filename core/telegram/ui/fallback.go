package ui

import tele "gopkg.in/telebot.v4"

// FallbackProvider exposes handlers used when incoming updates
// cannot be mapped to commands, callbacks, or expected documents.
type FallbackProvider interface {
	UnknownText() tele.HandlerFunc
	UnknownDocument() tele.HandlerFunc
	UnknownCallback() tele.HandlerFunc
}
