package fileLogger

import (
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
)

type Listener struct {
	Client *Client
	events chan interface{}
}

func (u Listener) Push(_ models.EventName, eventData interface{}) {
	u.events <- eventData
}

func (c *Client) eventsHandler(u Listener) {
	for event := range u.events {
		u.Listen("", event)
	}
}

func (u Listener) Listen(_ models.EventName, event interface{}) {
	switch event := event.(type) {
	case models.FileLoggerEvent:
		u.Client.Log(event.Src, event.Data, event.WithoutTime, event.ToDebug)
	default:
		log.Printf("registered an invalid fileLogger event: %T\n", event)
	}
}
