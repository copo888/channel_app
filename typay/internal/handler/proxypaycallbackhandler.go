package handler

import (
	"encoding/json"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/vaildx"
	"github.com/copo888/channel_app/typay/internal/logic"
	"github.com/copo888/channel_app/typay/internal/svc"
	"github.com/copo888/channel_app/typay/internal/types"
	"github.com/thinkeridea/go-extend/exnet"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

func ProxyPayCallBackHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		var req types.ProxyPayCallBackRequest

		//if err := httpx.ParseJsonBody(r, &req); err != nil {
		//	responsex.Json(w, r, responsex.FAIL, nil, err)
		//	return
		//}

		if err := httpx.ParseForm(r, &req); err != nil {
			responsex.Json(w, r, responsex.FAIL, nil, err)
			return
		}

		logx.WithContext(r.Context()).Infof("%+v", req)

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

		myIP := exnet.ClientIP(r)
		req.Ip = myIP

		l := logic.NewProxyPayCallBackLogic(r.Context(), ctx)
		resp, err := l.ProxyPayCallBack(&req)
		if err != nil {
			responsex.Json(w, r, err.Error(), resp, err)
		} else {
			w.Write([]byte(resp))
			//responsex.Json(w, r, responsex.SUCCESS, resp, err)
		}
	}
}
