package handler

import (
	"net/http"

	"github.com/copo888/channel_app/common/responsex"

	"github.com/copo888/channel_app/txpay/internal/logic"
	"github.com/copo888/channel_app/txpay/internal/svc"
	"go.opentelemetry.io/otel/trace"
)

func TxProxyPayQueryBalanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		l := logic.NewTxProxyPayQueryBalanceLogic(r.Context(), ctx)
		resp, err := l.TxProxyPayQueryBalance()
		if err != nil {
			responsex.Json(w, r, err.Error(), nil, err)
		} else {
			responsex.Json(w, r, responsex.SUCCESS, resp, err)
		}
	}
}
