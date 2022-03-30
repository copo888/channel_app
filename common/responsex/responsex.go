package responsex

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	_ "github.com/copo888/channel_app/locales"
	"github.com/neccohuang/easy-i18n/i18n"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/text/language"
	"net/http"
	"reflect"
)

type Body struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Trace   string      `json:"trace"`
}

func Json(w http.ResponseWriter, r *http.Request, code string, resp interface{}, err error) {
	var body Body

	span := trace.SpanFromContext(r.Context())

	i18n.SetLang(language.English)

	body.Code = code
	body.Message = i18n.Sprintf(code)
	if err != nil {
		if v, ok := err.(*errorx.Err); ok && v.GetMessage() != "" {
			body.Message += ": " + v.GetMessage()
			span.RecordError(errors.New(fmt.Sprintf("(%s)%s", code, v.GetMessage())))
		} else {
			span.RecordError(errors.New(fmt.Sprintf("(%s)%s %s", code, body.Message, err.Error())))
		}
	} else if !reflect.ValueOf(resp).IsNil() {
		body.Data = resp
	}
	body.Trace = span.SpanContext().TraceID().String()

	if responseBytes, err := json.Marshal(body); err == nil {
		span.SetAttributes(attribute.KeyValue{
			Key:   "response",
			Value: attribute.StringValue(string(responseBytes)),
		})
	}

	httpx.OkJson(w, body)
}
