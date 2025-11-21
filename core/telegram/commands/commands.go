package commands

import (
	tele "gopkg.in/telebot.v4"
)

// Command represents a bot command with its handler, description, and metadata.
type Command struct {
	Handler     tele.HandlerFunc
	Description string
	AdminOnly   bool
	Hidden      bool
	Aliases     []string
}
