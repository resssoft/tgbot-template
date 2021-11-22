package main

import (
	"github.com/resssoft/tgbot-template/internal/database"
	"github.com/resssoft/tgbot-template/internal/fileLogger"
	"github.com/resssoft/tgbot-template/internal/mediator"
	messenger "github.com/resssoft/tgbot-template/internal/messengers"
	"github.com/resssoft/tgbot-template/internal/models"
	routing "github.com/resssoft/tgbot-template/internal/webServer"
	"github.com/robfig/cron"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

type SystemListener struct{}

var onExit chan int

func main() {
	var err error
	onExit = make(chan int)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	dispatcher := mediator.NewDispatcher()
	if err := dispatcher.Register(
		SystemListener{},
		models.AppExit,
		models.SetLogDebugMode,
		models.SetLogInfoMode); err != nil {
		log.Info().Err(err).Send()
	}

	loggerClient := fileLogger.Provide(dispatcher)
	for filename, logName := range models.LogFiles {
		err = loggerClient.AddSource(filename, logName)
		if err != nil {
			log.Info().Err(err).Msgf("Error open log file %s", filename)
		}
	}
	time.Sleep(time.Second)
	defer loggerClient.CloseAll()

	_, err = database.ProvideMongo(dispatcher) //mongoDbApp
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	//userRep := repository.NewUserRepo(mongoDbApp)
	//leadRep := repository.NewLeadRepo(mongoDbApp)
	//pipeline.Provide(userRep, leadRep, dispatcher)

	go messenger.Initialize(dispatcher)
	go routing.NewRouter(dispatcher)

	log.Info().Msg("Prepare cron jobs")
	cronJobs := cron.New()
	// Every 6 hours
	err = cronJobs.AddFunc("0 0 */6 * * *", func() {
		log.Debug().Msg("========= START CRON ========= TASK AmoCrmRefreshToken")
		//log.Info().Err(dispatcher.Dispatch(models.AmoCrmRefreshToken, models.AmoCrmRefreshTokenEvent{})).Send()
	})
	err = cronJobs.AddFunc("0 */30 * * * *", func() {
		log.Debug().Msg("========= START CRON ========= TASK AmoCrmCronSleepersEvent")
		//log.Info().Err(dispatcher.Dispatch(models.AmoCrmCronSleepers, models.AmoCrmCronSleepersEvent{})).Send()
	})
	if err != nil {
		log.Err(err).Msg("cron err")
	}
	go cronJobs.Start()

	for code := range onExit {
		os.Exit(code)
	}
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
