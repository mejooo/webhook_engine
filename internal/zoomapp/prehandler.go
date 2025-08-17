package zoomapp

import (
	"encoding/json"
	"github.com/mejooo/webhook_engine/pkg/validators/zoom"

	"github.com/valyala/fasthttp"
)

type Middleware func(fasthttp.RequestHandler) fasthttp.RequestHandler
type wrapper struct{ mw Middleware }
func (w wrapper) Wrap(next fasthttp.RequestHandler) fasthttp.RequestHandler { return w.mw(next) }

// ZoomPreHandler intercepts Zoom CRC validation requests and responds immediately.
func ZoomPreHandler(_ any, _ Config) wrapper {
	return wrapper{mw: func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if string(ctx.Method()) == fasthttp.MethodPost && string(ctx.Path()) == "/webhook/zoom" {
				var req struct {
					Event   string `json:"event"`
					Payload struct{ PlainToken string `json:"plainToken"` } `json:"payload"`
				}
				body := ctx.PostBody()
				if json.Unmarshal(body, &req) == nil && req.Event == "endpoint.url_validation" && req.Payload.PlainToken != "" {
					secret, err := zoom.LoadSecretForCRC("")
					if err != nil { ctx.SetStatusCode(500); return }
					enc := zoom.EncryptPlainToken(secret, req.Payload.PlainToken)
					resp := struct {
						PlainToken     string `json:"plainToken"`
						EncryptedToken string `json:"encryptedToken"`
					}{req.Payload.PlainToken, enc}
					b, _ := json.Marshal(resp)
					ctx.Response.Header.SetContentType("application/json")
					ctx.SetStatusCode(200); ctx.SetBody(b)
					return
				}
			}
			next(ctx)
		}
	}}
}
