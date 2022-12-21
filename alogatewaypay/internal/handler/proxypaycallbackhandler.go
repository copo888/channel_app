package handler

import (
	"encoding/json"
	"encoding/xml"
	"github.com/copo888/channel_app/alogatewaypay/internal/logic"
	"github.com/copo888/channel_app/alogatewaypay/internal/svc"
	"github.com/copo888/channel_app/alogatewaypay/internal/types"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/vaildx"
	"github.com/thinkeridea/go-extend/exnet"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"net/http"
)

func ProxyPayCallBackHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		var req types.ProxyPayCallBackRequest
		bodyByte, err := ioutil.ReadAll(r.Body)
		if err := xml.Unmarshal(bodyByte, &req); err != nil {
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
