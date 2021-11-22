package fileLogger

import (
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
)

type Listener struct {
	Client *Client
}

func (u Listener) Listen(_ models.EventName, event interface{}) {
	switch event := event.(type) {
	case models.FileLoggerEvent:
		u.Client.Log(event.Src, event.Data, event.WithoutTime, event.ToDebug)
	default:
		log.Printf("registered an invalid fileLogger event: %T\n", event)
	}
}
