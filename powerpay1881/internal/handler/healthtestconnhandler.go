package handler

import (
	"github.com/copo888/channel_app/common/responsex"
	"net/http"

	"github.com/copo888/channel_app/powerpay1881/internal/logic"
	"github.com/copo888/channel_app/powerpay1881/internal/svc"
	"go.opentelemetry.io/otel/trace"
)

func HealthTestConnHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		l := logic.NewHealthTestConnLogic(r.Context(), ctx)
		resp, err := l.HealthTestConn()
		if err != nil {
			responsex.Json(w, r, err.Error(), nil, err)
		} else {
			w.Write([]byte(resp))
		}
	}
}
