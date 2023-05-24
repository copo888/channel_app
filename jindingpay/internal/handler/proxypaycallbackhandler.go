package handler

import (
	"encoding/json"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/jindingpay/internal/logic"
	"github.com/copo888/channel_app/jindingpay/internal/svc"
	"github.com/copo888/channel_app/jindingpay/internal/types"
	"github.com/thinkeridea/go-extend/exnet"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net/http"
)

func ProxyPayCallBackHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		var req types.ProxyPayCallBackRequest
		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			responsex.Json(w, r, responsex.FAIL, nil, err)
			return
		}

		logx.WithContext(r.Context()).Infof("PayOrder enter: %s", string(bodyBytes))

		if err = json.Unmarshal(bodyBytes, &req); err != nil {
			responsex.Json(w, r, err.Error(), nil, err)
			return
		}
		//if err := httpx.ParseJsonBody(r, &req); err != nil {
		//	responsex.Json(w, r, responsex.FAIL, nil, err)
		//	return
		//}
		//
		//logx.WithContext(r.Context()).Infof("%+v", req)
		//
		//if err := vaildx.Validator.Struct(req); err != nil {
		//	responsex.Json(w, r, responsex.INVALID_PARAMETER, nil, err)
		//	return
		//}
		//
		//if requestBytes, err := json.Marshal(req); err == nil {
		//	span.SetAttributes(attribute.KeyValue{
		//		Key:   "request",
		//		Value: attribute.StringValue(string(requestBytes)),
		//	})
		//}

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
