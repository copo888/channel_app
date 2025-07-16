package payutils

import (
	"context"
	"github.com/copo888/channel_app/bcpay/internal/svc"
	"github.com/copo888/channel_app/bcpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
)

func GetCryptoRate(exchangeInfo *types.ExchangeInfo, ctx *context.Context, svcCtx *svc.ServiceContext) (fiatAmount float64, err error) {
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
		Header("Authorization", "Bearer "+svcCtx.Config.AccessToken).
		Header("Content-type", "application/json").
		JSON(rateInfo)

	if ChnErr != nil {
		return 0, ChnErr
	}

	resp := struct {
		Amount   string `json:"amount, optional"`
		Currency string `json:"currency, optional"`
		message  string `json:"message, optional"`
	}{}

	if decodeErr := res.DecodeJSON(&resp); decodeErr != nil {
		return 0, decodeErr
	}

	orderAmount, errParse := strconv.ParseFloat(exchangeInfo.CryptoAmount, 64)
	if errParse != nil {
		return 0, errParse
	}
	rate, errParse := strconv.ParseFloat(resp.Amount, 64)
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
