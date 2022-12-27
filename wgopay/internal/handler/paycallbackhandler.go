package handler

import (
	"encoding/json"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/vaildx"
	"github.com/copo888/channel_app/wgopay/internal/logic"
	"github.com/copo888/channel_app/wgopay/internal/svc"
	"github.com/copo888/channel_app/wgopay/internal/types"
	"github.com/thinkeridea/go-extend/exnet"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net/http"
)

func PayCallBackHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		var req types.PayCallBackRequestX

		bodyBytes, err := io.ReadAll(r.Body)

		if err != nil {
			responsex.Json(w, r, responsex.DECODE_JSON_ERROR, nil, err)
			return
		}

		logx.WithContext(r.Context()).Infof("PayCallBack Enter: %s", string(bodyBytes))

		if err = json.Unmarshal(bodyBytes, &req); err != nil {
			responsex.Json(w, r, responsex.DECODE_JSON_ERROR, nil, err)
			return
		}

		if err = vaildx.Validator.Struct(req); err != nil {
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
		req.MyIp = myIP

		l := logic.NewPayCallBackLogic(r.Context(), ctx)
		resp, err := l.PayCallBack(&req)
		if err != nil {
			responsex.Json(w, r, err.Error(), nil, err)
		} else {
			w.Write([]byte(resp))
		}
	}
}
