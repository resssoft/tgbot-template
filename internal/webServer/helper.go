package routing

import (
	"encoding/json"
	"github.com/resssoft/tgbot-template/internal/models"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func CORS(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		log.Debug().Msgf("===========CORS============ %s %s \n%s",
			string(ctx.Request.Header.Method()),
			ctx.Request.URI().String(),
			string(ctx.PostBody()))
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "POST, GET")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		next(ctx)
	}
}

func getResponse(msg string, code int, status bool) models.Response {
	return models.Response{
		Error:  msg,
		Code:   code,
		Status: status,
	}
}

func writeJsonResponse(ctx *fasthttp.RequestCtx, code int, obj interface{}) {
	ctx.SetContentType("application/json; charset=utf8")
	ctx.Response.Header.SetCanonical(strContentType, strApplicationJSON)
	ctx.Response.SetStatusCode(code)
	if err := json.NewEncoder(ctx).Encode(obj); err != nil {
		log.Err(err).Send()
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}
