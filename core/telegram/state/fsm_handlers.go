package state

import tele "gopkg.in/telebot.v4"

var fsmHandlers = map[State]tele.HandlerFunc{}

// RegisterHandler associates a state with its handler.
func RegisterHandler(st State, h tele.HandlerFunc) {
	if h == nil {
		return
	}
	fsmHandlers[st] = h
}
