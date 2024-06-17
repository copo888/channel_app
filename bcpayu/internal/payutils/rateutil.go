package payutils

import (
	"context"
	"github.com/copo888/channel_app/bcpayu/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
)

func GetCryptoRate(exchangeInfo *types.ExchangeInfo, ctx *context.Context) (fiatAmount float64, err error) {
	span := trace.SpanFromContext(*ctx)
	rateInfo := struct {
		Command      string `json:"command"`
		Amount       string `json:"amount"`
		FromCurrency string `json:"from_currency"` //渠道汇率只能法币换虚拟币，不能虚拟币换虚拟币
		ToCurrency   string `json:"to_currency"`   //渠道汇率只能法币换虚拟币，不能虚拟币换虚拟币
		Type         string `json:"type"`
	}{
		Command:      "rates",
		Amount:       "100", //要转换加密货币的"法币数额" 预设100，换算要先除100再换算
		FromCurrency: exchangeInfo.Currency,
		ToCurrency:   exchangeInfo.Token,
		Type:         "BUY",
	}

	res, ChnErr := gozzle.Post(exchangeInfo.Url).Timeout(20).Trace(span).
		Header("Authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6IjZlNDRlMGU0YTE3NmY5OGMwYjUwZDNjNmJkMjQ0YWQzMzc5YzM3OWY0YjAwY2E3N2ZlYTQ1NjJmMWUxZjkxMGIyMGNjZDRjZTRkN2U1NzEyIn0.eyJhdWQiOiIxIiwianRpIjoiNmU0NGUwZTRhMTc2Zjk4YzBiNTBkM2M2YmQyNDRhZDMzNzljMzc5ZjRiMDBjYTc3ZmVhNDU2MmYxZTFmOTEwYjIwY2NkNGNlNGQ3ZTU3MTIiLCJpYXQiOjE3MTgzNDQxMzYsIm5iZiI6MTcxODM0NDEzNiwiZXhwIjoxNzE4OTQ4OTM2LCJzdWIiOiI0NDkiLCJzY29wZXMiOltdfQ.s11tFT2Ia9gL770mbKV8Y01PUlTPassDOXjPNhZIZ6CbUGFN_QQXAqSEdgeGfrucClN95HUNNNNYb9uQCDLvTaYmxBV0K9WfSrc_recZlHgmVJRHLk0ziTSPIQCavKK7kQKIBTDBcZuEzGU3XRJrql9m5uf9DPd4SchhsWAL4ZL3_pqgUiqGlPel8H9xxp2NsGcc1GwaxmD6O30qbdL5EDQsD3PmmD-c-VQjSkhmXVuZIcTW6HjkBgX7G9rexDejxR678V5WPJcmFVzBZISPXCO7GsW1tneUlBBYvgdsNX9W_IRv1g9CAEDHhugkoqHpLih6XgWvkG9wRLL4_51zSCZ4QbMAc_dP-ShNQm8lR-uqJsDbjmS-C9WplPzr3NzfD2H-sfZkwVeIAiIYgglzn8750f9G_qn1kQuQPCIOJaVgQzyUNnMHvfCupEdi0F48gi0wbKVohIM3jNbrJyknnfyWDh9a4jXNVao_GTQDlxuO56GpEO9Iwo9P8PWCUXllvBWIwaaoCMtqSG4Qrx4he2aExak4K-d-7C6Hghk4-bGU7WBt2TlgidGYpOV8q3XZOp4tKdpQ86MO7au_KCa3gXlY-nXVUGe79HhUukuGxC-UzK-ZinmxwSs6-GYaOEuuK9zvmkASknr4zo2TkVgsUaZE7fmfEDbB0i9bKbCcB4Q").
		Header("Content-type", "application/json").
		JSON(rateInfo)

	if ChnErr != nil {
		return 0, ChnErr
	}

	resp := struct {
		Rate     string `json:"amount"`
		Currency string `json:"currency"`
	}{}

	if decodeErr := res.DecodeJSON(&resp); decodeErr != nil {
		return 0, decodeErr
	}

	orderAmount, errParse := strconv.ParseFloat(exchangeInfo.CryptoAmount, 64)
	if errParse != nil {
		return 0, errParse
	}
	rate, errParse := strconv.ParseFloat(resp.Rate, 64)
	if errParse != nil {
		return 0, errParse
	}

	return orderAmount / (rate / 100), nil
}

type RateInfo struct {
	Command      string `json:"command"`
	Amount       string `json:"amount"`
	FromCurrency string `json:"from_currency"`
	ToCurrency   string `json:"to_currency"`
	Type         string `json:"type"`
}
