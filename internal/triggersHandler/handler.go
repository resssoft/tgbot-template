package pipeline

import (
	"encoding/json"
	"fmt"
	"github.com/resssoft/tgbot-template/internal/mediator"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/resssoft/tgbot-template/internal/repository"
	"github.com/rs/zerolog/log"
	"reflect"
	"strconv"
)

type Application interface {
	GoTo(int, string, models.Lead)
	DoQuestion(int, models.Lead)
	DoAnswer(string, models.Lead)
	CreateLead(models.PipelineLeadAddEvent) error
	NewMessage(event models.PipelineLeadAnswerEvent) error
	Parse([]byte) (models.SaleLogic, error)
	LeadChanged(models.PipelineLeadWebhookEvent) error
	AddConfig([]byte, int) error
}

const defaultPipeline = 35175244

type Client struct {
	Pipelines  map[int]models.SaleLogic
	UserRep    repository.UserRepository
	LeadRep    repository.LeadRepository
	dispatcher *mediator.Dispatcher
}

func Provide(userRep repository.UserRepository, leadRep repository.LeadRepository, dispatcher *mediator.Dispatcher) Application {
	client := &Client{
		UserRep:    userRep,
		LeadRep:    leadRep,
		dispatcher: dispatcher,
	}
	client.Pipelines = make(map[int]models.SaleLogic)
	if err := dispatcher.Register(
		Listener{
			Client: client,
		},
		models.PipelineEvents...); err != nil {
		log.Info().Err(err).Send()
	}
	return client
}

func (c *Client) GoTo(step int, stepType string, lead models.Lead) {
	log.Info().Msg("action goto")
	lead.Step = step
	//TODO: check err
	c.LeadRep.Update(lead.MongoID, lead)
	log.Info().Msgf("Change %s lead step to %v ", lead.User.TelegramUser.UserName, step)
	if stepType == "question" {
		c.DoQuestion(lead.Step, lead)
	}
	if stepType == "answer" {
		c.DoAnswer("", lead)
	}
}

func (c *Client) DoQuestion(step int, lead models.Lead) {
	currentModel, ok := c.Pipelines[lead.Pipeline]
	if !ok {
		return
	}
	blocks := currentModel.Blocks
	log.Info().Msgf("DoQuestion smd for %v [%v,%v]", step, len(blocks), len(blocks[strconv.Itoa(step)].Question))
	for _, q := range blocks[strconv.Itoa(step)].Question {
		log.Info().Msgf("handle question %s", q.Handler)
		switch q.Handler {
		case "show":
			log.Info().Msg("case show")
			switch q.Params.Type { //isEqual
			case "text":
				log.Info().Msg("case text")
				log.Info().Err(c.dispatcher.Dispatch(models.TelegramSendMessage, models.TelegramSendMessageEvent{
					ChatId:  lead.User.TelegramUser.ID,
					Message: q.Params.Value.(string),
				})).Send()
			case "buttons":
				log.Info().Msg("case buttons")
				log.Info().Err(c.dispatcher.Dispatch(models.TelegramSendButtons, models.TelegramSendButtonsEvent{
					ChatId:  lead.User.TelegramUser.ID,
					Message: q.Params.Value.(string),
					Buttons: q.Params.Buttons,
				})).Send()
			default:
				log.Info().Msg("nothing to show")
			}
		case "goto":
			c.GoTo(q.Params.Step, q.Params.Type.(string), lead)
		case "action":
			switch q.Params.Name {
			case "change_status":
				paramsJson, err := json.Marshal(q.Params.Params)
				if err != nil {
					log.Info().Err(err).Msg("params to json error")
					break
				}
				actionChangeStatusParam := new(models.ActionChangeStatusParam)
				err = json.Unmarshal(paramsJson, &actionChangeStatusParam)
				if err != nil {
					log.Info().Err(err).Msgf("Cant convert actionChangeStatusParam param: %v", paramsJson)
				} else {
					lead.Pipeline = actionChangeStatusParam.Value
				}
				c.DoQuestion(0, lead)
			}
		case "buttons":
			log.Info().Msgf("buttons %s", q.Params.Name)
		case "meta":
			log.Info().Msgf("meta %s", q.Params.Name)
		case "condition":
			log.Info().Msgf("condition %s", q.Params.Name)
		case "validations":
			log.Info().Msgf("validations %s", q.Params.Name)
		case "preset":
			log.Info().Msgf("preset %s", q.Params.Name)
		case "find":
			log.Info().Msgf("find %s", q.Params.Name)
		case "filter":
			log.Info().Msgf("filter %s", q.Params.Name)
		case "send_internal":
			log.Info().Msgf("send_internal %s", q.Params.Name)
		case "widget_request":
			log.Info().Msgf("widget_request %s", q.Params.Name)
		case "stop":
			log.Info().Msgf("stop %s", q.Params.Name)
		case "wait_answer":
			paramsJson, err := json.Marshal(q.Params)
			if err != nil {
				log.Info().Err(err).Msg("params to json error")
				break
			}
			actionChangeStatusParam := new(models.WaitAnswerParams)
			err = json.Unmarshal(paramsJson, &actionChangeStatusParam)
			if err != nil || actionChangeStatusParam == nil {
				log.Info().Err(err).Msgf("Cant convert actionChangeStatusParam param: src %v, Marshaled %v", q.Params, string(paramsJson))
			} else {
				log.Info().Interface("lead", lead).Msg("1")
				log.Info().Interface("actionChangeStatusParam", actionChangeStatusParam).Msg("2")
				lead.WaitAnswer.Step =
					actionChangeStatusParam.Step
				lead.WaitAnswer.Type = actionChangeStatusParam.Type
				lead.WaitAnswer.Enabled = true
				c.LeadRep.Update(lead.MongoID, lead)

				log.Info().Err(c.dispatcher.Dispatch(models.TelegramPromiseCreate, models.TelegramPromiseCreateEvent{
					User: lead.User.TelegramUser.ID,
				},
				)).Send()
			}
		default:
			log.Info().Msg("Unknown handler")
		}
	}
}

func (c *Client) DoAnswer(message string, lead models.Lead) {
	currentModel, ok := c.Pipelines[lead.Pipeline]
	if !ok {
		log.Error().Msgf("Answer: no stage exist for this lead %v", lead)
		return
	}
	blocks := currentModel.Blocks
	currentBlock, ok := blocks[strconv.Itoa(lead.Step)]
	if !ok {
		log.Error().Msgf("Answer: no step exist %v for lead %v", lead.Step, lead)
		return
	}
	log.Info().Msgf("DoAnswer smd for %v [%v,%v]", lead.Step, len(blocks), len(currentBlock.Answer))
	log.Info().Interface("******", currentBlock.Answer).Send()
	for qi, _ := range blocks {
		log.Info().Msgf("DoAnswer [%s,%s,%v]", qi, strconv.Itoa(lead.Step), qi == strconv.Itoa(lead.Step))
	}
	for _, q := range currentBlock.Answer {
		log.Info().Msgf("handle Answer %s", q.Handler)
		switch q.Handler {
		case "action":
			log.Info().Msgf("action %s", q.Params)
		case "buttons":
			typeParams := reflect.TypeOf(q.Params).Kind()
			switch typeParams {
			case reflect.Slice:
				paramsList := reflect.ValueOf(q.Params)
				log.Info().Msgf("buttons reflect.Slice %v", paramsList)
				for i := 0; i < paramsList.Len(); i++ {
					param := paramsList.Index(i).Interface().(models.BQAParams)
					if param.Value == message {
						for _, innerParam := range param.Params {
							switch innerParam.Handler {
							case "goto":
								log.Info().Msgf("goto Answer")
								log.Info().Interface("innerParam", innerParam.Params).Send()
								log.Info().Interface("innerParam", fmt.Sprintf("%v", innerParam.Params)).Send()
								paramsJson, err := json.Marshal(innerParam.Params)
								if err != nil {
									log.Info().Err(err).Msg("params to json error")
									break
								}
								log.Info().Interface("innerParam", paramsJson).Send()
								log.Info().Interface("innerParam", fmt.Sprintf("%v", string(paramsJson))).Send()

								gotoValue := new(models.GotoParam)
								err = json.Unmarshal(paramsJson, &gotoValue)
								if err != nil {
									log.Info().Err(err).Msgf("Cant convert Goto param: %v", paramsJson)
								} else {
									c.GoTo(gotoValue.Step, gotoValue.Type, lead)
								}
							default:
								log.Info().Interface("innerParam", innerParam).Send()
							}
						}
					}
				}
			default:
				log.Info().Msgf("Unknown param type %v", typeParams)
			}
		default:
			log.Info().Msg("Unknown handler")
		}
	}
}

func (c *Client) Parse(config []byte) (models.SaleLogic, error) {
	data := new(map[string]models.Block)
	err := json.Unmarshal(config, &data)
	return models.SaleLogic{
		Blocks: *data,
	}, err
}

func (c *Client) NewMessage(data models.PipelineLeadAnswerEvent) error {
	lead, err := c.LeadRep.GetByMessengerID(data.User.ID, data.Messenger)
	if lead.WaitAnswer.Enabled {
		lead.WaitAnswer.Enabled = false
		lead.Step = lead.WaitAnswer.Step
		c.LeadRep.Update(lead.MongoID, lead)
		switch lead.WaitAnswer.Type {
		case "question":
			go c.DoQuestion(lead.WaitAnswer.Step, lead)
		case "answer":
			go c.DoAnswer(data.Message, lead)
		}
		return nil
	}
	if err == nil {
		go c.DoAnswer(data.Message, lead)
	} else {
		log.Info().Err(err).Msg("Lead not found")
	}
	return nil
}

func (c *Client) CreateLead(data models.PipelineLeadAddEvent) error {
	//TODO: move this functional to webhooks after integration
	log.Info().Msgf("Add lead %v", data.User.ID)
	//check if user exist
	user, _ := c.UserRep.Add(models.User{
		TelegramUser: data.User,
	})
	//check exist lead
	lead, _ := c.LeadRep.Add(models.Lead{
		User:     user,
		Source:   data.Source,
		Step:     0,
		Pipeline: defaultPipeline,
	})
	c.DoQuestion(0, lead)
	return nil
}

func isEqual(variable interface{}, intVal int, strVal string) bool {
	switch val := variable.(type) {
	case string:
		return val == strVal
	case int:
		return val == intVal
	default:
		return false
	}
}

func (c *Client) LeadChanged(data models.PipelineLeadWebhookEvent) error {
	for _, leadStatus := range data.Data.Leads.Status {
		lead, err := c.LeadRep.GetByMessengerID(leadStatus.Id, "telegram") //TODO: fix to GetByAmoCrmID after integration
		if err != nil {
			log.Info().Err(err).Send()
		}
		log.Info().Int64("For lead", lead.User.TelegramUser.ID).Int("PP", leadStatus.PipelineId).Msg("LeadChanged")
		lead.Pipeline = leadStatus.PipelineId
		c.LeadRep.Update(lead.MongoID, lead)
		c.DoQuestion(0, lead)
	}
	return nil
}

func (c *Client) AddConfig(config []byte, pipelineId int) error {
	if len(config) != 0 {
		model, err := c.Parse(config)
		if err != nil {
			log.Info().Err(err).Msg("json err")
		} else {
			c.Pipelines[pipelineId] = model
			log.Info().Interface("model", model).Send() //TODO: remove later
		}
	} else {
		log.Error().Msg("config  is empty")
	}
	return nil
}
