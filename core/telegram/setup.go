package telegram

import tele "gopkg.in/telebot.v4"

// SetupCommands registers bot command menu and related scopes.
func SetupCommands(bot *tele.Bot, reg *Registry) {
	InitBotCommands(bot, reg)
}
