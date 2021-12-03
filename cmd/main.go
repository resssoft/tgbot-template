package main

import (
	"github.com/resssoft/tgbot-template/internal/announcementManager"
	"github.com/resssoft/tgbot-template/internal/database"
	"github.com/resssoft/tgbot-template/internal/fileLogger"
	"github.com/resssoft/tgbot-template/internal/mediator"
	messenger "github.com/resssoft/tgbot-template/internal/messengers"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/resssoft/tgbot-template/internal/repository"
	routing "github.com/resssoft/tgbot-template/internal/webServer"
	"github.com/robfig/cron"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

var onExit chan int

type SystemListener struct {
	events chan models.EventName
}

func (u SystemListener) Push(eventName models.EventName, _ interface{}) {
	u.events <- eventName
}

func (u SystemListener) Listen(eventName models.EventName, _ interface{}) {
	switch eventName {
	case models.AppExit:
		onExit <- 0
	case models.SetLogDebugMode:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case models.SetLogInfoMode:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func eventsHandler(u SystemListener) {
	for event := range u.events {
		u.Listen(event, nil)
	}
}

func main() {
	var err error
	onExit = make(chan int)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	systemListener := SystemListener{}
	dispatcher := mediator.NewDispatcher()
	if err := dispatcher.Register(
		systemListener,
		models.SystemEvents...); err != nil {
		log.Info().Err(err).Send()
	}
	go eventsHandler(systemListener)

	loggerClient := fileLogger.Provide(dispatcher)
	for filename, logName := range models.LogFiles {
		err = loggerClient.AddSource(filename, logName)
		if err != nil {
			log.Info().Err(err).Msgf("Error open log file %s", filename)
		}
	}
	time.Sleep(time.Second)
	defer loggerClient.CloseAll()

	mongoDbApp, err := database.ProvideMongo(dispatcher)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	postRep := repository.NewPostRepo(mongoDbApp)
	announcementManager.Provide(dispatcher, postRep)

	go messenger.Initialize(dispatcher)
	go routing.NewRouter(dispatcher)

	log.Info().Msg("Prepare cron jobs")
	cronJobs := cron.New()
	// Every 10 minutes
	err = cronJobs.AddFunc("0 */2 * * * *", func() {
		log.Debug().Msg("========= START CRON ========= TASK announcementManager")
		log.Info().Err(dispatcher.Dispatch(models.AnManagerCron, models.AnManagerEvent{})).Send()
	})
	if err != nil {
		log.Err(err).Msg("cron err")
	}
	go cronJobs.Start()

	for code := range onExit {
		os.Exit(code)
	}
}
