package pipeline

import (
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
)

type Listener struct {
	Client *Client
}

func (u Listener) Listen(_ models.EventName, event interface{}) {
	switch event := event.(type) {
	case models.PipelineLeadAddEvent:
		u.Client.CreateLead(event)
	case models.PipelineLeadAnswerEvent:
		u.Client.NewMessage(event)
	case models.PipelineLeadWebhookEvent:
		u.Client.LeadChanged(event)
	case models.PipelineConfigUploadEvent:
		u.Client.AddConfig(event.Config, event.PipelineId)
	default:
		log.Printf("registered an invalid pipeline event: %T\n", event)
	}
}
