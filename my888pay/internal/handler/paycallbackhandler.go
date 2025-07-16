package handler

import (
	"encoding/json"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/vaildx"
	"github.com/copo888/channel_app/my888pay/internal/logic"
	"github.com/copo888/channel_app/my888pay/internal/payutils"
	"github.com/copo888/channel_app/my888pay/internal/svc"
	"github.com/copo888/channel_app/my888pay/internal/types"
	"github.com/thinkeridea/go-extend/exnet"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"net/http"
)

func PayCallBackHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		span := trace.SpanFromContext(r.Context())
		defer span.End()

		var req types.PayCallBackRequest

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			responsex.Json(w, r, responsex.DECODE_JSON_ERROR, nil, err)
			return
		}
		defer r.Body.Close()
		//if err := httpx.ParseJsonBody(r, &req); err != nil {
		//	responsex.Json(w, r, responsex.DECODE_JSON_ERROR, nil, err)
		//	return
		//}

		// Form 格式
		//if err := httpx.ParseForm(r, &req); err != nil {
		//	responsex.Json(w, r, responsex.FAIL, nil, err)
		//	return
		//}

		logx.WithContext(r.Context()).Infof("%s", string(body))

		key, err := payutils.DecodeBase64Key(ctx.Config.HashKey)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("密钥解码失败:", err)
		}
		decryptText, err := payutils.Decrypt(string(body), key)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("解密失败:", err)
		}

		if err := json.Unmarshal([]byte(decryptText), &req); err != nil {
			logx.WithContext(r.Context()).Errorf("解密失败:", err)
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
