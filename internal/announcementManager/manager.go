package announcementManager

import (
	"github.com/resssoft/tgbot-template/internal/mediator"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/resssoft/tgbot-template/internal/repository"
	"github.com/rs/zerolog/log"
	"time"
)

const (
	eventsBuffer = 500
)

type App interface {
	GetTable(string, string, string, string) string
}

type Client struct {
	dispatcher *mediator.Dispatcher
	postRep    repository.PostRepository
}

func Provide(dispatcher *mediator.Dispatcher, postRep repository.PostRepository) App {
	client := &Client{
		dispatcher: dispatcher,
		postRep:    postRep,
	}
	listener := Listener{
		Client: client,
		events: make(chan interface{}, eventsBuffer),
	}
	if err := dispatcher.Register(
		listener,
		models.AnnouncementManagerEvents...); err != nil {
		log.Info().Err(err).Send()
	}
	go func() {
		time.Sleep(time.Second)
		log.Info().Err(client.dispatcher.Dispatch(models.TelegramCommands, models.TelegramCommandsEvent{
			Commands: commands,
		})).Send()
	}()
	go client.eventsHandler(listener)
	return client
}

func (c *Client) GetTable(dateStart, dateEnd, src, sheetId string) string {
	return ""
}
