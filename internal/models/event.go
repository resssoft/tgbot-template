package models

const FileLogFatal = "fatal"
const FileLogErrors = "errors"
const FileLogContacts = "contacts"
const FileLogWebHooks = "webHooks"
const FileLogRequests = "requests"
const FileLogMessenger = "messenger"

var LogFiles = map[string]string{
	"fatal.txt":     FileLogFatal,
	"errors.txt":    FileLogErrors,
	"contacts.txt":  FileLogContacts,
	"webHooks.txt":  FileLogWebHooks,
	"requests.txt":  FileLogRequests,
	"messenger.txt": FileLogMessenger,
}

type EventName string

const NothingEvent EventName = "nothing"

const TelegramSendMessage EventName = "messenger.telegram.message.send"
const TelegramSendImage EventName = "messenger.telegram.image.send"
const TelegramSendButtons EventName = "messenger.telegram.buttons.send"
const TelegramWebHook EventName = "messenger.telegram.webhook"
const TelegramPromiseCreate EventName = "messenger.telegram.promise.create"
const TelegramProvideMessage EventName = "messenger.telegram.provide"

var TelegramEvents = []EventName{
	TelegramSendMessage,
	TelegramSendButtons,
	TelegramWebHook,
	TelegramPromiseCreate,
	TelegramSendImage,
	TelegramProvideMessage,
}

type TelegramResponse struct {
	Data []byte
}

const SetLogDebugMode EventName = "log.mode.debug"
const SetLogInfoMode EventName = "log.mode.info"

const AppExit EventName = "app.exit"

const LogToFile EventName = "fileLogger.log.data"

var FileLoggerEvents = []EventName{
	LogToFile,
}

type Listener interface {
	Listen(eventName EventName, event interface{})
}

type Job struct {
	EventName EventName
	EventType interface{}
}

type TelegramSendMessageEvent struct {
	ChatId   int64
	Message  string
	ImageURL string
	MsgId    string
	ToFile   bool
	Buttons  []string
}

type TelegramProvideMessageEvent struct {
	ChatId    int64    `json:"chatId,omitempty"`
	Message   string   `json:"message,omitempty"`
	ImageURL  string   `json:"imageURL,omitempty"`
	MsgId     string   `json:"msgId,omitempty"`
	ToFile    bool     `json:"toFile,omitempty"`
	Buttons   []string `json:"buttons,omitempty"`
	ContactId string   `json:"contactId,omitempty"`
	LeadId    string   `json:"leadId,omitempty"`
}

type TelegramSendButtonsEvent struct {
	ChatId  int64
	Message string
	Buttons []string
	MsgId   string
}

type UpdateUserTelegramDataEvent struct {
	Email string
	User  TelegramUser
}

type TelegramPromiseCreateEvent struct {
	User int64
}

type FileLoggerEvent struct {
	Src         string
	Data        string
	WithoutTime bool
	ToDebug     bool
}

type Secure struct {
	Hash   string     `json:"hash,omitempty"`
	Filter UserFilter `json:"filter,omitempty"`
}

type UserFilter struct {
	ChatId       int64  `json:"chatId,omitempty"`
	UserName     string `json:"username,omitempty"`
	ContactId    int    `json:"contactId,omitempty"`
	LeadId       int    `json:"leadId,omitempty"`
	Conversation string `json:"conversation,omitempty"`
	Source       string `json:"source,omitempty"`
	BlockedBot   int64  `json:"blockedBot,omitempty"`
}
