package messenger

import (
	"github.com/resssoft/tgbot-template/internal/mediator"
	"github.com/resssoft/tgbot-template/internal/messengers/telegram"
)

func Initialize(dispatcher *mediator.Dispatcher) {
	telegramApp, tgErr := telegram.Initialize(dispatcher)
	if tgErr == nil {
		go telegramApp.MessageHandler()
	}
}
