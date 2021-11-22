package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	config "github.com/resssoft/tgbot-template/configuration"
	"github.com/rs/zerolog/log"
	"net/http"
)

var commandsInfo = map[string]commandInfo{
	"ver": {
		names:       []string{"/ver", "/version"},
		description: "Get app version",
		isPublic:    true,
	},
	"this": {
		names:       []string{"/this"},
		description: "Get user ID and chat ID",
		isPublic:    true,
	},
	"info": {
		names:       []string{"/info"},
		description: "Get app info",
		permissions: []int64{config.TelegramAdminId(), config.TelegramReportChatId()},
	},
}

func (t *tgConfig) commandsHandler() {
	log.Info().Msg("start commandsHandler")
	for command := range t.commands {
		update := command.Data
		chatId := update.Message.Chat.ID
		log.Info().Msg("Handle command")
		switch {
		case t.CheckCommand(command.Name, 0, "/start"):
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "menu")
			msg.ReplyMarkup = menu
			t.Send(msg)

		case t.CheckCommand(command.Name, 0, commandsInfo["ver"].names...):
			log.Info().Msg("version")
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, config.Version)
			t.Send(msg)

		case t.CheckCommand(command.Name, 0, commandsInfo["this"].names...):
			log.Info().Msg("this")
			statistic := fmt.Sprintf("Current chat: %v \nFrom %v",
				command.Data.Message.Chat.ID,
				command.Data.Message.From.ID)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, statistic)
			t.Send(msg)

		case t.CheckCommand(command.Name, 0, "/test"):
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "menu")
			msg.ReplyMarkup = menu
			t.Send(msg)

		case t.CheckCommand(command.Name, chatId, commandsInfo["info"].names...):
			appStat, _ := json.MarshalIndent(config.GetMemUsage(), "", "    ")
			t.Send(tgbotapi.NewMessage(update.Message.Chat.ID, string(appStat)))

		case t.CheckCommand(command.Name, chatId, "/eventTest"):
			//log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmRefreshToken, models.AmoCrmRefreshTokenEvent{})).Send()
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "event sent")
			t.Send(msg)

		case t.CheckCommand(command.Name, chatId, "/import"):
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
			/*
				log.Info().Err(t.dispatcher.Dispatch(models.AmoCrmImportContacts, models.AmoCrmImportContactsEvent{
					Data:   buf.Bytes(),
					ChatId: update.Message.Chat.ID,
				})).Send()
			*/
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "import downloaded")
			t.Send(msg)

		default:
			log.Info().Msgf("unhanding message %s", update.Message.Text)
		}

		//TODO: save to file or db
	}
}
