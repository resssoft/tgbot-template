package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	menuSetLogDebugMode = "menuSetLogDebugMode"
	menuSetLogInfoMode  = "menuSetLogInfoMode"
	menuHelp            = "help"
)

var menu = tgbotapi.NewInlineKeyboardMarkup(
	[]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Set Log Debug Mode", menuSetLogDebugMode),
		tgbotapi.NewInlineKeyboardButtonData("Set Log Info Mode", menuSetLogInfoMode),
	},
	[]tgbotapi.InlineKeyboardButton{},
	[]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Help", menuHelp),
	},
)

type UserPromise struct {
	userID       int64
	toKeepOne    bool
	promiseType  string
	KeyboardOpen bool
}
type Command struct {
	Name   string
	Params string
	Data   tgbotapi.Update
}

type TgFileInfo struct {
	Ok     bool `json:"ok,omitempty"`
	Result struct {
		FileId       string `json:"file_id,omitempty"`
		FileUniqueId string `json:"file_unique_id,omitempty"`
		FileSize     int    `json:"file_size,omitempty"`
		FilePath     string `json:"file_path,omitempty"`
	} `json:"result,omitempty"`
}
