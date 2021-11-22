package telegram

import (
	"bytes"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	config "github.com/resssoft/tgbot-template/configuration"
	"github.com/resssoft/tgbot-template/internal/mediator"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const (
	workers            = 4
	timeout        int = 60
	updateOffset       = 0
	parseMode          = "MarkdownV2"
	maxConnections     = 10
	updatesBuffer      = 1000
	commandsBuffer     = 300
)

type TgApp interface {
	MessageHandler()
	SendButtons(int64, string, []string) (tgbotapi.Message, error)
}

type tgConfig struct {
	BotTelegram  *tgbotapi.BotAPI
	dispatcher   *mediator.Dispatcher
	userPromises map[int64]UserPromise
	users        map[int64]models.TelegramUser
	BotName      string
	updates      chan tgbotapi.Update
	commands     chan Command
}

func Initialize(dispatcher *mediator.Dispatcher) (TgApp, error) {
	botTelegram, err := tgbotapi.NewBotAPI(config.TelegramToken())
	if err != nil {
		log.Printf("\ntgbotapi problem " + err.Error())
		return nil, err
	}
	log.Printf("Init telegram bot %s", botTelegram.Self.UserName)
	tgApp := &tgConfig{
		BotTelegram:  botTelegram,
		dispatcher:   dispatcher,
		userPromises: make(map[int64]UserPromise),
		users:        make(map[int64]models.TelegramUser),
		BotName:      botTelegram.Self.UserName,
		updates:      make(chan tgbotapi.Update, updatesBuffer),
		commands:     make(chan Command, commandsBuffer),
	}
	config.SetTelegramAdminBot(tgApp.BotName)
	if err := dispatcher.Register(
		Listener{
			tgApp: tgApp,
		}, models.TelegramEvents...); err != nil {
		log.Info().Err(err).Send()
	}
	for i := 0; i < workers; i++ {
		go tgApp.commandsHandler()
	}
	return tgApp, nil
}

func (t *tgConfig) Response(update tgbotapi.Update) {
	t.updates <- update
}

func (t *tgConfig) GetUpdatesChannel() tgbotapi.UpdatesChannel {
	return t.updates
}

func (t *tgConfig) CheckCommand(msg string, chatId int64, commands ...string) bool {
	result := false
	for _, command := range commands {
		if msg == command || msg == fmt.Sprintf("%s@%s", command, t.BotName) {
			result = true
			break
		}
	}
	if chatId != 0 {
		if !(chatId == config.TelegramAdminId() || chatId == config.TelegramReportChatId()) {
			result = false
		}
	}
	return result
}

func (t *tgConfig) Send(message tgbotapi.Chattable) (tgbotapi.Message, error) {
	//TODO: override sending
	return t.BotTelegram.Send(message)
}

func (t *tgConfig) SendMessage(event models.TelegramSendMessageEvent) (tgbotapi.Message, error) {
	var messageResult tgbotapi.Message
	var err error
	var message tgbotapi.Chattable
	chatId := config.TelegramAdminId()
	if event.ChatId != 0 {
		chatId = event.ChatId
	}
	if event.Message != "" {
		message = tgbotapi.NewMessage(chatId, event.Message)
	}
	if event.ImageURL != "" {
		response, err := http.Get(event.ImageURL)
		if err != nil {
			log.Info().Err(err).Msgf("download photo error from link %s", event.ImageURL)
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(response.Body)
		file := tgbotapi.FileBytes{
			Name:  "photo.jpg",
			Bytes: buf.Bytes(),
		}
		urlLower := strings.ToLower(event.ImageURL)
		switch {
		case strings.Contains(urlLower, ".gif"),
			strings.Contains(urlLower, ".mp4"):
			message = tgbotapi.NewVideoUpload(chatId, file)
		case strings.Contains(urlLower, ".doc"),
			strings.Contains(urlLower, ".txt"),
			strings.Contains(urlLower, ".pdf"):
			message = tgbotapi.NewDocumentUpload(chatId, file)
		default:
			message = tgbotapi.NewPhotoUpload(chatId, file)
		}
	} else {
		if event.ToFile {
			log.Info().Msg("event.ToFile")
			file := tgbotapi.FileBytes{
				Name:  fmt.Sprintf("import-%s.txt", time.Now().Format(config.DateTimeFormat)),
				Bytes: []byte(event.Message),
			}
			message = tgbotapi.NewDocumentUpload(chatId, file)
		} else {
			message = tgbotapi.NewMessage(chatId, event.Message)
		}
	}
	switch msg := message.(type) {
	case tgbotapi.MessageConfig:
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		msg.Text = event.Message
		message = msg
	case tgbotapi.VideoConfig:
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		if event.Message != "" {
			msg.Caption = event.Message
		} else {
			msg.Caption = "-\n"
		}
		message = msg
	case tgbotapi.PhotoConfig:
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		if event.Message != "" {
			msg.Caption = event.Message
		}
		message = msg
	case tgbotapi.DocumentConfig:
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		if event.Message != "" && !event.ToFile {
			msg.Caption = event.Message
		} else {
			msg.Caption = "-\n"
		}
		message = msg
	}
	message = t.addButtonsIfExist(message, event.Buttons)

	messageResult, err = t.Send(message)

	if !event.ToFile {
		t.dispatcher.Dispatch(models.LogToFile, models.FileLoggerEvent{
			Src: models.FileLogMessenger,
			Data: fmt.Sprintf("Tg message to: %v, text: %s, buttons: %v, file: %s, result %v",
				chatId,
				event.Message,
				event.Buttons,
				event.ImageURL,
				messageResult,
			),
		})
	}
	if err != nil {
		log.Info().Err(err).Msg("send message error")
		if "Forbidden: user is deactivated" != err.Error() ||
			"Forbidden: bot was blocked by the user" != err.Error() {
			message := tgbotapi.NewMessage(chatId, event.Message)
			messageResult, err = t.Send(message)
		}
	}
	return messageResult, err
}

func (t *tgConfig) addButtonsIfExist(message tgbotapi.Chattable, buttons []string) tgbotapi.Chattable {
	if len(buttons) > 0 {
		var rows [][]tgbotapi.KeyboardButton
		for _, bText := range buttons {
			rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(bText)))
		}
		keyboard := tgbotapi.NewReplyKeyboard(rows...)
		keyboard.OneTimeKeyboard = true
		switch msg := message.(type) {
		case tgbotapi.MessageConfig:
			msg.ReplyMarkup = keyboard
			message = msg
		case tgbotapi.VideoConfig:
			msg.ReplyMarkup = keyboard
			message = msg
		case tgbotapi.PhotoConfig:
			msg.ReplyMarkup = keyboard
			message = msg
		case tgbotapi.DocumentConfig:
			msg.ReplyMarkup = keyboard
			message = msg
		}
	}
	return message
}

func (t *tgConfig) ErrorHandler(messageResult tgbotapi.Message, err error) {
	log.Info().Interface("messageResult", messageResult).Msg("tg Message ErrorHandler")
	if err == nil {
		return
	}
	if err.Error() == "Forbidden: bot was blocked by the user" {

	}
	if err.Error() == "Forbidden: user is deactivated" {

	}
}

func (t *tgConfig) SendButtons(chatId int64, msg string, buttons []string) (tgbotapi.Message, error) {
	tgMsg := tgbotapi.NewMessage(chatId, msg)
	var rows [][]tgbotapi.KeyboardButton
	for _, bText := range buttons {
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(bText)))
	}
	keyboard := tgbotapi.NewReplyKeyboard(rows...)
	keyboard.OneTimeKeyboard = true
	tgMsg.ReplyMarkup = keyboard
	result, err := t.BotTelegram.Send(tgMsg)
	if err != nil {
		log.Error().Err(err).Msg("Bot message with buttons sending error")
	}
	localPromise := UserPromise{
		userID:       chatId,
		toKeepOne:    false,
		KeyboardOpen: true,
	}

	if false {
		//TODO: add thread safe writes by mutex.Lock()
		t.userPromises[chatId] = localPromise
	}
	//TODO: add thread safe writes by mutex.Lock()
	//tgUser, userExist := t.users[chatId]
	for _, bText := range buttons {
		msg += fmt.Sprintf("\n[%s]", bText)
	}
	/*
		if config.AmoCrmEventFromBotMessage() {
			log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmMessageSend, models.AmoCrmMessageSendEvent{
				Message:      msg,
				IsBot:        true,
				TelegramUser: models.TelegramUser{ID: chatId},
			})).Send()
		}
	*/
	return result, err
}

func (t *tgConfig) MessageHandler() {
	log.Printf("\nStart spy tg messages")

	if t.BotTelegram == nil {
		log.Printf("\nBotTelegram pointer is nil")
		return
	}

	updates := t.GetUpdatesChannel()
	if config.TelegramCallBackUrl() != "" {
		/*
			log.Debug().Msgf("Set telegram webhook to %s", config.TelegramCallBackUrl())
			urlObject, _ := url.Parse(config.TelegramCallBackUrl())
			_, err := t.BotTelegram.SetWebhook(tgbotapi.WebhookConfig{
				URL:            urlObject,
				MaxConnections: maxConnections,
			})
			if err != nil {
				log.Info().Err(err).Msgf("telegtam webhook install error")
				return
			}
			webhookInfo, err := t.BotTelegram.GetWebhookInfo()
			if err != nil {
				log.Info().Err(err).Msg("telegtam webhook install error")
			}
			if webhookInfo.LastErrorDate != 0 {
				log.Printf("Telegram callback failed: %s", webhookInfo.LastErrorMessage)
			}
		*/
	} else {
		_, err := t.BotTelegram.RemoveWebhook()
		if err != nil {
			log.Info().Err(err).Msg("telegtam Remove Webhook error")
		}
		log.Debug().Msgf("Set telegram updates with timeout %v", timeout)
		u := tgbotapi.NewUpdate(updateOffset)
		u.Timeout = timeout
		updates, _ = t.BotTelegram.GetUpdatesChan(u)

		if config.TelegramAdminId() != 0 {
			msg := tgbotapi.NewMessage(config.TelegramReportChatId(), "Start "+config.AppName()+" v"+config.Version)
			t.Send(msg)
		}
	}

	for update := range updates {
		if update.Message == nil && update.InlineQuery != nil {
			continue
		}
		//messageProcessed = false

		//work with menu
		log.Debug().Interface("TG message", update).Send()

		if update.Message != nil {
			if update.Message.Chat.ID != config.TelegramReportChatId() &&
				int64(update.Message.From.ID) != update.Message.Chat.ID &&
				config.TelegramExitOtherGroups() {
				t.BotTelegram.LeaveChat(tgbotapi.ChatConfig{ChatID: update.Message.Chat.ID})
				continue
			}
		}
		if update.CallbackQuery != nil {
			log.Debug().Interface("CallbackQuery", update.CallbackQuery).Send()
			/*
				if config.AmoCrmEventFromBotMessage() {
					log.Info().Err(t.dispatcher.Dispatch(models.PipelineLeadAnswer, models.PipelineLeadAnswerEvent{
						Message:   update.CallbackQuery.Data,
						Messenger: "telegram",
						User: models.TelegramUser{
							ID:           update.CallbackQuery.Message.Chat.ID,
							FirstName:    update.CallbackQuery.Message.From.FirstName,
							LastName:     update.CallbackQuery.Message.From.LastName,
							UserName:     update.CallbackQuery.Message.From.UserName,
							LanguageCode: update.CallbackQuery.Message.From.LanguageCode,
							IsBot:        update.CallbackQuery.Message.From.IsBot,
						},
					})).Send()
				}
			*/
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.Command())
			t.Send(msg)
		} else {
			log.Info().Msg("Handle updates")
			if update.Message == nil {
				continue
			}
			message := ""
			if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {
				message = update.Message.Text
			} else {
				message = update.Message.Caption
			}
			log.Info().Msgf("Handle updates: %s", message)

			t.dispatcher.Dispatch(models.LogToFile, models.FileLoggerEvent{
				Src: models.FileLogMessenger,
				Data: fmt.Sprintf("Tg message from: %v %s %s %s, text %s",
					update.Message.From.ID,
					update.Message.From.UserName,
					update.Message.From.FirstName,
					update.Message.From.LastName,
					message,
				),
			})

			ParsedCommands, commandValue := splitCommand(message, " ")
			log.Debug().Msg(commandValue)

			t.commands <- Command{
				Name:   ParsedCommands[0],
				Parsed: ParsedCommands,
				Params: commandValue,
				Data:   update,
			}

			//userPromises
			/*

				tgUser := models.TelegramUser{
					ID:           update.Message.Chat.ID,
					FirstName:    update.Message.From.FirstName,
					LastName:     update.Message.From.LastName,
					UserName:     update.Message.From.UserName,
					LanguageCode: update.Message.From.LanguageCode,
					IsBot:        update.Message.From.IsBot,
				}
				//TODO: add thread safe writes by mutex.Lock()
				t.users[update.Message.Chat.ID] = tgUser
				if _, ok := t.userPromises[update.Message.Chat.ID]; ok {
					log.Info().Msg("handle Promises")
					delete(t.userPromises, update.Message.Chat.ID)
					continue
				}
			*/
		}
	}
}

func splitCommand(command string, separate string) ([]string, string) {
	if command == "" {
		return []string{}, ""
	}
	if separate == "" {
		separate = " "
	}
	result := strings.Split(command, separate)
	return result, strings.Replace(command, result[0]+separate, "", -1)
}
