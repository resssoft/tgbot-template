package routing

import (
	"github.com/buaazp/fasthttprouter"
	config "github.com/resssoft/tgbot-template/configuration"
	"github.com/resssoft/tgbot-template/internal/mediator"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"net/http"
)

var (
	strContentType     = []byte("Content-Type")
	strApplicationJSON = []byte("application/json")
)

type routerData struct {
	dispatcher *mediator.Dispatcher
}

func NewRouter(dispatcher *mediator.Dispatcher) error {
	routerDataConf := &routerData{dispatcher: dispatcher}
	router := fasthttprouter.New()
	router.GET(config.WebServerPrefix()+"/", IndexHandler)
	router.GET(config.WebServerPrefix()+"/version", VersionHandler)
	router.POST(config.WebServerPrefix()+config.TelegramCallBackUri(), routerDataConf.telegram)
	log.Info().Msg("Start web server by " + config.WebServerAddress())
	return fasthttp.ListenAndServe(config.WebServerAddress(), CORS(router.Handler))
}

func IndexHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.SetStatusCode(403)
}

func VersionHandler(ctx *fasthttp.RequestCtx) {
	writeJsonResponse(ctx, http.StatusOK, map[string]string{"version": config.Version})
}

func (r *routerData) telegram(ctx *fasthttp.RequestCtx) {
	log.Debug().Str("tg event", string(ctx.PostBody())).Send()
	log.Info().Err(
		r.dispatcher.Dispatch(
			models.TelegramWebHook,
			models.TelegramResponse{
				Data: ctx.PostBody(),
			}),
	).Send()
	writeJsonResponse(ctx, http.StatusOK, getResponse("", 0, true))
}
