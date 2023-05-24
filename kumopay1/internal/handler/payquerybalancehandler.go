package handler

import (
	"github.com/copo888/channel_app/common/responsex"
	"net/http"

	"github.com/copo888/channel_app/kumopay1/internal/logic"
	"github.com/copo888/channel_app/kumopay1/internal/svc"
	"go.opentelemetry.io/otel/trace"
)

func PayQueryBalanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		l := logic.NewPayQueryBalanceLogic(r.Context(), ctx)
		resp, err := l.PayQueryBalance()
		if err != nil {
			responsex.Json(w, r, err.Error(), nil, err)
		} else {
			responsex.Json(w, r, responsex.SUCCESS, resp, err)
		}
	}
}
