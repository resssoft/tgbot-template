package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	config "github.com/resssoft/tgbot-template/configuration"
)

func (t *tgConfig) commandsHandler() {
	for command := range t.commands {
		switch command.Name {
		case "ver", "version":
			msg := tgbotapi.NewMessage(command.Data.Message.Chat.ID, config.Version)
			t.Send(msg)
		case "ver", "version":
			msg := tgbotapi.NewMessage(command.Data.Message.Chat.ID, config.Version)
			t.Send(msg)
		}
	}
}
