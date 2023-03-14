package handler

import (
	"github.com/copo888/channel_app/common/responsex"
	"net/http"

	"github.com/copo888/channel_app/u2pay/internal/logic"
	"github.com/copo888/channel_app/u2pay/internal/svc"
	"go.opentelemetry.io/otel/trace"
)

func ProxyPayQueryBalanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		l := logic.NewProxyPayQueryBalanceLogic(r.Context(), ctx)
		resp, err := l.ProxyPayQueryBalance()
		if err != nil {
			responsex.Json(w, r, err.Error(), nil, err)
		} else {
			responsex.Json(w, r, responsex.SUCCESS, resp, err)
		}
	}
}
