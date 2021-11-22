package configuration

import (
	"fmt"
	"github.com/hako/durafmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"runtime"
	"strings"
	"time"
)

const (
	Version        = "0.0.1" //grepVersion
	DateTimeFormat = "2006-01-02 15:04:05 -0700"
)

var startTime = time.Now()
var startTimeString = time.Now().Format(DateTimeFormat)

type AppStatus struct {
	MemoryUsage  AppMemoryUsage
	NumGoroutine int
	NumCPU       int
	NumCgoCall   int64
	GoVersion    string
	Version      string
	Server       serverInfo
}

type AppMemoryUsage struct {
	Alloc        string `json:"Alloc"`
	TotalAlloc   string `json:"TotalAlloc"`
	HeapAlloc    string `json:"HeapAlloc"`
	HeapReleased string `json:"HeapReleased"`
	Sys          string `json:"Sys"`
	NumGC        string `json:"NumGC"`
	LastGC       string `json:"LastGC"`
}

type serverInfo struct {
	Goos        string `json:"goos"`
	ServerTime  string `json:"serverTime"`
	ServerStart string `json:"serverStart"`
	Uptime      string `json:"serverUptime"`
	MainChat    string `json:"MainChat"`
	LogChat     string `json:"LogChat"`
	Pipeline    string `json:"pipeline"`
	LeadStatus  string `json:"leadStatus"`
	LeadStatusB string `json:"leadStatusBot"`
}

func init() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AllowEmptyEnv(true)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Info().Msg("Unable to read config file")
	}
}

func AppName() string {
	return viper.GetString("name")
}

func WebServerAddress() string {
	return viper.GetString("server.url")
}

func WebServerPrefix() string {
	return viper.GetString("server.prefix")
}

func MongoUrl() string {
	return viper.GetString("db.mongoDb.url")
}

func MongoDbName() string {
	return viper.GetString("db.mongoDb.dbname")
}

func TelegramToken() string {
	return viper.GetString("telegram.token")
}

func TelegramAdminId() int64 {
	return viper.GetInt64("telegram.admin")
}

func TelegramExitOtherGroups() bool {
	return viper.GetBool("telegram.exitOtherGroups")
}

func SetTelegramAdminBot(name string) {
	viper.Set("telegram.bot", name)
}

func TelegramReportChatId() int64 {
	return viper.GetInt64("telegram.chat")
}

func TelegramCallBackUrl() string {
	return viper.GetString("telegram.callBackUrl")
}

func TelegramCallBackUri() string {
	return viper.GetString("telegram.callBackUri")
}

func AmoCrmSecret() string {
	return viper.GetString("amocrm.clientSecret")
}

func AmoCrmClientId() string {
	return viper.GetString("amocrm.clientId")
}

func AmoCrmRedirectUri() string {
	return viper.GetString("amocrm.redirectUri")
}

func AmoCrmAmojoId() string {
	return viper.GetString("amocrm.amojoId")
}

func AmoCrmChannelSecret() string {
	return viper.GetString("amocrm.channel.secretKey")
}

func AmoCrmScopeId() string {
	return viper.GetString("amocrm.channel.scopeId")
}

func AmoCrmUri() string {
	return viper.GetString("amocrm.uri")
}

func AmoCrmPipeline() int {
	return viper.GetInt("amocrm.pipeline")
}

func SetAmoCrmPipeline(value int) {
	viper.Set("amocrm.pipeline", value)
}

func AmoCrmLeadStatus() int {
	return viper.GetInt("amocrm.leadStatus")
}

func SetAmoCrmLeadStatus(value int) {
	viper.Set("amocrm.leadStatus", value)
}

func AmoCrmLeadStatusBot() int {
	return viper.GetInt("amocrm.leadStatusBot")
}

func SetAmoCrmLeadStatusBot(value int) {
	viper.Set("amocrm.leadStatusBot", value)
}

func AmoCrmEventFromBotMessage() bool {
	return viper.GetBool("amocrm.sendCrmEventFromBotMessage")
}

func LogPath() string {
	return viper.GetString("log.path")
}

func GetFacebookPixelId() string {
	return viper.GetString("facebook.pixelId")
}

func GetFacebookAccessToken() string {
	return viper.GetString("facebook.accessToken")
}

func GetMemUsage() AppStatus {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	lastGC := time.Unix(0, int64(m.LastGC)).Format(DateTimeFormat)
	return AppStatus{
		MemoryUsage: AppMemoryUsage{
			Alloc:      fmt.Sprintf("%v MiB", bToMb(m.Alloc)),
			TotalAlloc: fmt.Sprintf("%v MiB", bToMb(m.TotalAlloc)),
			Sys:        fmt.Sprintf("%v MiB", bToMb(m.Sys)),
			HeapAlloc:  fmt.Sprintf("%v MiB", bToMb(m.HeapAlloc)),
			NumGC:      fmt.Sprintf("%v", m.NumGC),
			LastGC:     lastGC,
		},
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		NumCgoCall:   runtime.NumCgoCall(),
		GoVersion:    runtime.Version(),
		Version:      Version,
		Server: serverInfo{
			Goos:        runtime.GOOS,
			ServerTime:  time.Now().Format(DateTimeFormat),
			ServerStart: startTimeString,
			Uptime:      durafmt.Parse(time.Now().Sub(startTime)).LimitFirstN(2).String(),
			MainChat:    fmt.Sprintf("%v", TelegramAdminId()),
			LogChat:     fmt.Sprintf("%v", TelegramReportChatId()),
			Pipeline:    fmt.Sprintf("%v", AmoCrmPipeline()),
			LeadStatus:  fmt.Sprintf("%v", AmoCrmLeadStatus()),
			LeadStatusB: fmt.Sprintf("%v", AmoCrmLeadStatusBot()),
		},
	}
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
