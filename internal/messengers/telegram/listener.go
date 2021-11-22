package telegram

import (
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	config "github.com/resssoft/tgbot-template/configuration"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
)

type Listener struct {
	tgApp *tgConfig
}

func (u Listener) Listen(_ models.EventName, event interface{}) {
	switch event := event.(type) {
	case models.TelegramResponse:
		// Handle data from http callback
		var update tgbotapi.Update
		err := json.Unmarshal(event.Data, &update)
		if err != nil {
			log.Info().Err(err).Msg("Telegram Response err")
		} else {
			u.tgApp.Response(update)
		}
	case models.TelegramSendMessageEvent:
		u.tgApp.ErrorHandler(u.tgApp.SendMessage(event))
	case models.TelegramSendButtonsEvent:
		chatId := config.TelegramAdminId()
		if event.ChatId != 0 {
			chatId = event.ChatId
		}
		u.tgApp.SendButtons(chatId, event.Message, event.Buttons)
	case models.TelegramPromiseCreateEvent:
		u.tgApp.userPromises[event.User] = UserPromise{
			userID:      event.User,
			toKeepOne:   false,
			promiseType: "",
		}
	case models.TelegramProvideMessageEvent:
		chatId := event.ChatId
		if event.ContactId != "" {
			//TODO: get by contact
		}
		if event.LeadId != "" {
			//TODO: get by lead
		}
		u.tgApp.ErrorHandler(u.tgApp.SendMessage(models.TelegramSendMessageEvent{
			ChatId:   chatId,
			Message:  event.Message,
			ImageURL: event.ImageURL,
			MsgId:    event.MsgId,
			ToFile:   event.ToFile,
			Buttons:  event.Buttons,
		}))
	default:
		log.Printf("registered an invalid telegram event: %T\n", event)
	}
}
