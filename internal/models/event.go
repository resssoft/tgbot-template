package models

import "net/url"

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

const TelegramSendMessage EventName = "messenger.telegram.message.send"
const TelegramSendImage EventName = "messenger.telegram.image.send"
const TelegramSendButtons EventName = "messenger.telegram.buttons.send"
const TelegramWebHook EventName = "messenger.telegram.webhook"
const TelegramPromiseCreate EventName = "messenger.telegram.promise.create"
const TelegramProvideMessage EventName = "messenger.telegram.provide"
const TelegramDuplicateMessage EventName = "messenger.telegram.duplicate"

var TelegramEvents = []EventName{
	TelegramSendMessage,
	TelegramSendButtons,
	TelegramWebHook,
	TelegramPromiseCreate,
	TelegramSendImage,
	TelegramProvideMessage,
	TelegramDuplicateMessage,
}

type TelegramResponse struct {
	Data []byte
}

const PipelineLeadAdd EventName = "pipeline.lead.add"
const PipelineLeadAnswer EventName = "pipeline.lead.answer"
const PipelineLeadWebhook EventName = "pipeline.lead.webhook"
const PipelineConfigUpload EventName = "pipeline.config.upload"

var PipelineEvents = []EventName{
	PipelineLeadAdd,
	PipelineLeadAnswer,
	PipelineLeadWebhook,
	PipelineConfigUpload,
}

const AmoCrmWebhookAuth EventName = "amocrm.webhook.auth"
const AmoCrmMessageSend EventName = "amocrm.message.send"
const AmoCrmFileSend EventName = "amocrm.file.send"
const AmoCrmWebhookChat EventName = "amocrm.webhook.chat"
const AmoCrmRefreshToken EventName = "amocrm.token.refresh"
const AmoCrmFillContacts EventName = "amocrm.contacts.fill"
const AmoCrmSendInfo EventName = "amocrm.info.send"
const AmoCrmSendContacts EventName = "amocrm.contacts.send"
const AmoCrmRemoveContacts EventName = "amocrm.contacts.remove"
const AmoCrmAccessToken EventName = "amocrm.token.access"
const AmoCrmUserData EventName = "amocrm.data.user"
const AmoCrmImportContacts EventName = "amocrm.data.import"
const AmoCrmWebhookLeadStatus EventName = "amocrm.webhook.lead.status"
const AmoCrmCronSleepers EventName = "amocrm.cron.sleepers"

var AmoCrmEvents = []EventName{
	AmoCrmWebhookAuth,
	AmoCrmMessageSend,
	AmoCrmWebhookChat,
	AmoCrmFileSend,
	AmoCrmRefreshToken,
	AmoCrmFillContacts,
	AmoCrmSendInfo,
	AmoCrmSendContacts,
	AmoCrmRemoveContacts,
	AmoCrmAccessToken,
	AmoCrmUserData,
	AmoCrmImportContacts,
	AmoCrmWebhookLeadStatus,
	AmoCrmCronSleepers,
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

type AmoCrmCronSleepersEvent struct{}

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

type TelegramDuplicateMessageEvent struct {
	Chat string
	Data url.Values
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

type PipelineLeadAddEvent struct {
	Source    string
	Messenger string
	User      TelegramUser
}

type PipelineLeadAnswerEvent struct {
	Message   string
	Messenger string
	User      TelegramUser
}

type PipelineLeadWebhookEvent struct {
	Data AmoCrmChangePipeline
}

type PipelineConfigUploadEvent struct {
	PipelineId int
	Config     []byte
}

type TelegramPromiseCreateEvent struct {
	User int64
}

type AmoCrmOAuthEvent struct {
	Code     string
	Referer  string
	ClientId string
	Widget   string
}

type AmoCrmMessageSendEvent struct {
	Message      string
	IsBot        bool
	TelegramUser TelegramUser
	Source       string
}

type AmoCrmMessageChatRequestEvent struct {
	Data string
}

type AmoCrmMessageSendFileEvent struct {
	MessageMedia AmoCrmMMessageMedia
	TelegramUser TelegramUser
	Message      string
}

type AmoCrmRefreshTokenEvent struct{}

type AmoCrmAccessTokenEvent struct{}

type AmoCrmFillContactsEvent struct{}

type AmoCrmSendInfoEvent struct {
	ChatId int64
}

type AmoCrmSendContactsEvent struct {
	TelegramId   string
	TelegramName string
	ContactId    string
	LeadId       string
	Source       string
}

type AmoCrmImportContactsEvent struct {
	Data   []byte
	ChatId int64
}

type AmoCrmRemoveContactsEvent struct {
	TelegramId   string
	TelegramName string
	ContactId    string
	LeadId       string
}

type AmoCrmMMessageMedia struct {
	Type     string
	Media    string
	FileName string
	FileSize int
}

type FileLoggerEvent struct {
	Src         string
	Data        string
	WithoutTime bool
	ToDebug     bool
}

// TODO: remove later
type AmoCrmCheckRandomsEvent struct {
	ChatId int64
	Count  int
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

type AmoCrmUserDataEvent struct {
	ChanData *chan interface{}
	Filter   UserFilter
}

type ResponseUserInfo struct {
	Data  []Lead `json:"data"`
	Error error  `json:"error"`
	Count int    `json:"count"`
}

type AmoCrmWebhookLeadStatusEvent struct {
	Data string
}
