package handler

import (
	"encoding/json"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/common/vaildx"
	"github.com/copo888/channel_app/dpay/internal/logic"
	"github.com/copo888/channel_app/dpay/internal/svc"
	"github.com/copo888/channel_app/dpay/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

func ProxyPayOrderQueryHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		var req types.ProxyPayOrderQueryRequest

		if err := httpx.ParseJsonBody(r, &req); err != nil {
			responsex.Json(w, r, responsex.FAIL, nil, err)
			return
		}

		if err := vaildx.Validator.Struct(req); err != nil {
			responsex.Json(w, r, responsex.INVALID_PARAMETER, nil, err)
			return
		}

		if requestBytes, err := json.Marshal(req); err == nil {
			span.SetAttributes(attribute.KeyValue{
				Key:   "request",
				Value: attribute.StringValue(string(requestBytes)),
			})
		}

		// 驗證密鑰
		authenticationProxykey := r.Header.Get("authenticationProxykey")
		if isOK, err := utils.MicroServiceVerification(authenticationProxykey, ctx.Config.ApiKey.ProxyKey, ctx.Config.ApiKey.PublicKey); err != nil || !isOK {
			err = errorx.New(responsex.INTERNAL_SIGN_ERROR)
			responsex.Json(w, r, err.Error(), nil, err)
			return
		}

		l := logic.NewProxyPayOrderQueryLogic(r.Context(), ctx)
		resp, err := l.ProxyPayOrderQuery(&req)
		if err != nil {
			responsex.Json(w, r, err.Error(), nil, err)
		} else {
			responsex.Json(w, r, responsex.SUCCESS, resp, err)
		}
	}
}
