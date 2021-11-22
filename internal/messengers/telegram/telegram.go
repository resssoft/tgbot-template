package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	config "github.com/resssoft/tgbot-template/configuration"
	"github.com/resssoft/tgbot-template/internal/mediator"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	timeout        int = 60
	updateOffset       = 0
	parseMode          = "MarkdownV2"
	maxConnections     = 10
	updatesBuffer      = 1000
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
	}
	config.SetTelegramAdminBot(tgApp.BotName)
	if err := dispatcher.Register(
		Listener{
			tgApp: tgApp,
		}, models.TelegramEvents...); err != nil {
		log.Info().Err(err).Send()
	}
	return tgApp, nil
}

func (t *tgConfig) Response(update tgbotapi.Update) {
	t.updates <- update
}

func (t *tgConfig) GetUpdatesChannel() tgbotapi.UpdatesChannel {
	return t.updates
}

func (t *tgConfig) IsCommand(msg, command string) bool {
	return msg == command || msg == fmt.Sprintf("%s@%s", command, t.BotName)
}

func (t *tgConfig) Send(message tgbotapi.Chattable) (tgbotapi.Message, error) {
	switch msg := message.(type) {
	case tgbotapi.MessageConfig:
		//TODO: add thread safe writes by mutex.Lock()
		//tgUser, userExist := t.users[msg.ChatID]
		if config.AmoCrmEventFromBotMessage() {
			log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmMessageSend, models.AmoCrmMessageSendEvent{
				Message:      msg.Text,
				IsBot:        true,
				TelegramUser: models.TelegramUser{ID: msg.ChatID},
			})).Send()
		}
	}
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

	if config.AmoCrmEventFromBotMessage() {
		//TODO: add thread safe writes by mutex.Lock()
		t.userPromises[chatId] = localPromise
	}
	//TODO: add thread safe writes by mutex.Lock()
	//tgUser, userExist := t.users[chatId]
	for _, bText := range buttons {
		msg += fmt.Sprintf("\n[%s]", bText)
	}
	if config.AmoCrmEventFromBotMessage() {
		log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmMessageSend, models.AmoCrmMessageSendEvent{
			Message:      msg,
			IsBot:        true,
			TelegramUser: models.TelegramUser{ID: chatId},
		})).Send()
	}
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
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.Command())
			t.Send(msg)
		} else {
			if update.Message == nil {
				continue
			}
			if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {

				t.dispatcher.Dispatch(models.LogToFile, models.FileLoggerEvent{
					Src: models.FileLogMessenger,
					Data: fmt.Sprintf("Tg message from: %v %s %s %s, text %s",
						update.Message.From.ID,
						update.Message.From.UserName,
						update.Message.From.FirstName,
						update.Message.From.LastName,
						update.Message.Text,
					),
				})

				splitedCommands, commandValue := splitCommand(update.Message.Text, " ")
				log.Debug().Msg(commandValue)
				commandsCount := len(splitedCommands)
				commandName := splitedCommands[0]

				if update.Message.Chat.Type != "private" {
					log.Info().Msgf("Group detected %s %v", update.Message.Chat.Type, update.Message.Chat.ID)
				}

				if t.IsCommand(commandName, "/ver") {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, config.Version)
					t.Send(msg)
					continue
				}

				// TODO: remove later
				if t.IsCommand(commandName, "/CheckRandoms") {
					countForRandom := 50
					if len(splitedCommands) >= 2 {
						countForRandom, _ = strconv.Atoi(splitedCommands[1])
					}
					t.dispatcher.Dispatch(models.AmoCrmSendInfo, models.AmoCrmCheckRandomsEvent{
						ChatId: update.Message.Chat.ID,
						Count:  countForRandom})
					continue
				}

				if t.IsCommand(commandName, "/this") {
					statistic := fmt.Sprintf("Current chat: %v \nFrom %v",
						update.Message.Chat.ID,
						update.Message.From.ID)
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, statistic)
					t.Send(msg)
					continue
				}

				if t.IsCommand(commandName, "/test") {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "test")
					msg.ReplyMarkup = menu
					t.Send(msg)
					continue
				}

				t.dispatcher.Dispatch(models.TelegramDuplicateMessage, models.TelegramDuplicateMessageEvent{
					Chat: strconv.Itoa(update.Message.From.ID),
					Data: url.Values{
						"mode":    {"post"},
						"prov":    {"0"},
						"lastMod": {"0"},
						"name": {fmt.Sprintf("%s %s (@%s) %v",
							update.Message.From.FirstName,
							update.Message.From.LastName,
							update.Message.From.UserName,
							update.Message.Chat.ID,
						)},
						"text": {update.Message.Text},
					},
				})

				if update.Message.Chat.ID == config.TelegramAdminId() ||
					update.Message.Chat.ID == config.TelegramReportChatId() {
					if t.IsCommand(commandName, "/info") {
						appStat, _ := json.MarshalIndent(config.GetMemUsage(), "", "    ")
						t.Send(tgbotapi.NewMessage(update.Message.Chat.ID, string(appStat)))
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmSendInfo, models.AmoCrmSendInfoEvent{
							ChatId: update.Message.Chat.ID,
						})).Send()
						continue
					}

					if t.IsCommand(commandName, "/refreshToken") {
						//log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmRefreshToken, models.AmoCrmRefreshTokenEvent{})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}

					if t.IsCommand(commandName, "/setPipeline") {
						newPipeline, err := strconv.Atoi(commandValue)
						oldPipeline := config.AmoCrmPipeline()
						messageText := "Pipeline not changed"
						if err == nil {
							messageText = fmt.Sprintf("Pipeline change from %v to %v", oldPipeline, newPipeline)
							config.SetAmoCrmPipeline(newPipeline)
						}
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, messageText)
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/setLeadStatus") {
						newLeadStatus, err := strconv.Atoi(commandValue)
						oldLeadStatus := config.AmoCrmLeadStatus()
						messageText := "Lead Status not changed"
						if err == nil {
							messageText = fmt.Sprintf("Lead Status change from %v to %v", oldLeadStatus, newLeadStatus)
							config.SetAmoCrmLeadStatus(newLeadStatus)
						}
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, messageText)
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/fillContacts") {
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmFillContacts, models.AmoCrmFillContactsEvent{})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/logPromises") {
						log.Info().Msgf("userPromises: count %v list: \n %#v", len(t.userPromises), t.userPromises)
						continue
					}
					if t.IsCommand(commandName, "/contactsByTG") {
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmSendContacts, models.AmoCrmSendContactsEvent{
							TelegramId: commandValue,
						})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/export") {
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmSendContacts, models.AmoCrmSendContactsEvent{
							TelegramId: "1",
						})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/cronSleepers") {
						log.Info().Err(t.dispatcher.Dispatch(
							models.AmoCrmCronSleepers,
							models.AmoCrmCronSleepersEvent{})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}

					if t.IsCommand(commandName, "/contactsByTGLogin") {
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmSendContacts, models.AmoCrmSendContactsEvent{
							TelegramName: commandValue,
						})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/contactsByContact") {
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmSendContacts, models.AmoCrmSendContactsEvent{
							ContactId: commandValue,
						})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/contactsByLead") {
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmSendContacts, models.AmoCrmSendContactsEvent{
							LeadId: commandValue,
						})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/contactsBySource") {
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmSendContacts, models.AmoCrmSendContactsEvent{
							Source: commandValue,
						})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}
					if t.IsCommand(commandName, "/contactRemove") {
						log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmRemoveContacts, models.AmoCrmRemoveContactsEvent{
							ContactId: commandValue,
						})).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
						t.Send(msg)
						continue
					}
				}
				tgUser := models.TelegramUser{
					ID:           update.Message.Chat.ID,
					FirstName:    update.Message.From.FirstName,
					LastName:     update.Message.From.LastName,
					UserName:     update.Message.From.UserName,
					LanguageCode: update.Message.From.LanguageCode,
					IsBot:        update.Message.From.IsBot,
				}
				log.Debug().Interface("update.Message.From", update.Message.From).Send()
				log.Debug().Interface("update.Message.Chat", update.Message.Chat).Send()
				messageText := update.Message.Text
				if update.Message.ReplyToMessage != nil {
					replyFrom := update.Message.ReplyToMessage.From.FirstName +
						" " + update.Message.ReplyToMessage.From.LastName
					if update.Message.ReplyToMessage.From.UserName != "" {
						replyFrom += " @" + update.Message.ReplyToMessage.From.UserName
					}
					messageText += fmt.Sprintf(
						"\n\nReply from [%s] to message:\n%s",
						replyFrom,
						update.Message.ReplyToMessage.Text)
				}
				if commandsCount == 0 {
					continue
				}
				srcTag := ""
				if commandName == "/start" {
					srcTag = commandValue
				}
				log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmMessageSend, models.AmoCrmMessageSendEvent{
					Message:      messageText,
					IsBot:        false,
					TelegramUser: tgUser,
					Source:       srcTag,
				})).Send()
				//log.Printf("[%s, %n] %s", update.Message.Chat.UserName, update.Message.Chat.ID, update.Message.Text)
				//log.Printf("%#v", t.userPromises)

				if config.AmoCrmEventFromBotMessage() {
					//TODO: add thread safe writes by mutex.Lock()
					t.users[update.Message.Chat.ID] = tgUser
					if _, ok := t.userPromises[update.Message.Chat.ID]; ok {
						log.Info().Msg("handle Promises")
						log.Info().Err(t.dispatcher.Dispatch(models.PipelineLeadAnswer, models.PipelineLeadAnswerEvent{
							Message:   update.Message.Text,
							Messenger: "telegram",
							User: models.TelegramUser{
								ID:           update.Message.Chat.ID,
								FirstName:    update.Message.From.FirstName,
								LastName:     update.Message.From.LastName,
								UserName:     update.Message.From.UserName,
								LanguageCode: update.Message.From.LanguageCode,
								IsBot:        update.Message.From.IsBot,
							},
						})).Send()

						delete(t.userPromises, update.Message.Chat.ID)
						continue
					}
				}

				switch commandName {
				case "/start":
					if config.AmoCrmEventFromBotMessage() {
						if commandValue != "" {
							log.Info().Err(t.dispatcher.Dispatch(models.PipelineLeadAdd, models.PipelineLeadAddEvent{
								Source:    commandValue,
								Messenger: "telegram",
								User:      tgUser,
							})).Send()
						}
					}

				default:
					//msg := tgbotapi.NewMessage(update.Message.Chat.ID, "This is unsupported command.")
					//msg.ReplyToMessageID = update.Message.MessageID
					//t.Send(msg)
				}
			} else {
				if update.Message.Chat.ID == config.TelegramAdminId() &&
					(t.IsCommand(update.Message.Caption, "/import") || t.IsCommand(update.Message.Text, "/import")) {
					log.Info().Msg("import!!!")
					if update.Message.Document == nil {
						log.Info().Msg("empty document")
						continue
					}
					fileId := update.Message.Document.FileID
					response, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s",
						config.TelegramToken(), fileId))
					if err != nil {
						log.Info().Err(err).Msgf("download TG import file error")
						continue
					}
					buf := new(bytes.Buffer)
					buf.ReadFrom(response.Body)
					result := buf.String()
					if response.StatusCode >= 300 {
						log.Debug().Str("tg import file some error ", result).Send()
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, result)
						t.Send(msg)
					}
					fileInfo := TgFileInfo{}
					err = json.Unmarshal([]byte(result), &fileInfo)
					if err != nil {
						log.Info().Err(err).Msg("Decode fileInfo err")
						continue
					}

					fileUrl := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s",
						config.TelegramToken(), fileInfo.Result.FilePath)

					response, err = http.Get(fileUrl)
					if err != nil {
						log.Info().Err(err).Msgf("download TG import file error")
						continue
					}
					buf = new(bytes.Buffer)
					buf.ReadFrom(response.Body)

					log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmImportContacts, models.AmoCrmImportContactsEvent{
						Data:   buf.Bytes(),
						ChatId: update.Message.Chat.ID,
					})).Send()

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "import downloaded")
					t.Send(msg)
					continue
				}
				tgUser := models.TelegramUser{
					ID:           update.Message.Chat.ID,
					FirstName:    update.Message.From.FirstName,
					LastName:     update.Message.From.LastName,
					UserName:     update.Message.From.UserName,
					LanguageCode: update.Message.From.LanguageCode,
					IsBot:        update.Message.From.IsBot,
				}
				fileId := ""
				var messageMedia models.AmoCrmMMessageMedia
				switch {
				case update.Message.Photo != nil:
					messageMedia.Type = "picture"
					for _, photoItem := range *update.Message.Photo {
						fileId = photoItem.FileID
						messageMedia.FileSize = photoItem.FileSize
					}
				case update.Message.Sticker != nil:
					messageMedia.Type = "sticker"
					fileId = update.Message.Sticker.FileID
					messageMedia.FileSize = update.Message.Sticker.FileSize
				case update.Message.Video != nil:
					messageMedia.Type = "video"
					fileId = update.Message.Video.FileID
					messageMedia.FileSize = update.Message.Video.FileSize
				case update.Message.Voice != nil:
					messageMedia.Type = "voice"
					fileId = update.Message.Voice.FileID
					messageMedia.FileSize = update.Message.Voice.FileSize
				case update.Message.Audio != nil:
					messageMedia.Type = "audio"
					fileId = update.Message.Audio.FileID
					messageMedia.FileSize = update.Message.Audio.FileSize
				case update.Message.Document != nil:
					messageMedia.Type = "file"
					fileId = update.Message.Document.FileID
					messageMedia.FileSize = update.Message.Document.FileSize
					messageMedia.FileName = update.Message.Document.FileName
				case update.Message.VideoNote != nil:
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "unsupported command")
					t.Send(msg)
					continue
				case update.Message.Animation != nil:
					messageMedia.Type = "file"
					fileId = update.Message.Animation.FileID
					messageMedia.FileSize = update.Message.Animation.FileSize
					messageMedia.FileName = update.Message.Animation.FileName
				case update.Message.Venue != nil:
					log.Info().Interface("tg file ignore Venue", update).Send()
				case update.Message.Contact != nil:
					log.Info().Interface("tg file ignore Contact", update).Send()
					log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmMessageSend, models.AmoCrmMessageSendEvent{
						Message: fmt.Sprintf("Contact %s %s [%v] phone %s",
							update.Message.Contact.FirstName,
							update.Message.Contact.LastName,
							update.Message.Contact.UserID,
							update.Message.Contact.PhoneNumber,
						),
						TelegramUser: tgUser,
					})).Send()
				case update.Message.Location != nil:
					log.Info().Str("tg file Location", fmt.Sprintf("%f.5 %f.5", update.Message.Location.Longitude, update.Message.Location.Latitude)).Send()
					log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmMessageSend, models.AmoCrmMessageSendEvent{
						Message: fmt.Sprintf("Location %f.5 %f.5",
							update.Message.Location.Longitude,
							update.Message.Location.Latitude),
						TelegramUser: tgUser,
					})).Send()
				}
				if fileId == "" {
					log.Info().Interface("tg file info not found", update).Send()
					continue
				}

				response, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s",
					config.TelegramToken(), fileId))
				if err != nil {
					log.Info().Err(err).Msgf("download TG photo error")
				}
				buf := new(bytes.Buffer)
				buf.ReadFrom(response.Body)
				result := buf.String()
				log.Debug().Str("tg fileInfo unparsed", result).Send()
				fileInfo := TgFileInfo{}
				err = json.Unmarshal([]byte(result), &fileInfo)
				if err != nil {
					log.Info().Err(err).Msg("Decode fileInfo err")
					continue
				}
				messageMedia.Media = fmt.Sprintf("https://api.telegram.org/file/bot%s/%s",
					config.TelegramToken(), fileInfo.Result.FilePath)
				log.Info().Interface("fileInfo", fileInfo).Send()
				log.Info().Interface("messageMedia", messageMedia).Send()
				log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmFileSend, models.AmoCrmMessageSendFileEvent{
					MessageMedia: messageMedia,
					TelegramUser: tgUser,
					Message:      update.Message.Caption,
				})).Send()
			}
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
