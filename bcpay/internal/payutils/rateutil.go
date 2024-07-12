package payutils

import (
	"context"
	"github.com/copo888/channel_app/bcpay/internal/types"
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
		Header("Authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6IjYxYWRhNzM3ZWNmMzIwMjE3ZmVlYzUyZDIzNDgyNTkwOTI1YjAyNjI2YzY1MjAwMDk4ODc1ZmY2NzI2N2FkMjdkNGY3MmQ5NmVkYzgzNDY4In0.eyJhdWQiOiIxIiwianRpIjoiNjFhZGE3MzdlY2YzMjAyMTdmZWVjNTJkMjM0ODI1OTA5MjViMDI2MjZjNjUyMDAwOTg4NzVmZjY3MjY3YWQyN2Q0ZjcyZDk2ZWRjODM0NjgiLCJpYXQiOjE3MjA2ODcxMjQsIm5iZiI6MTcyMDY4NzEyNCwiZXhwIjoxNzIxMjkxOTI0LCJzdWIiOiI0NDkiLCJzY29wZXMiOltdfQ.loWg5juLro4FtuwOY-5ui_D1CQ_IcCZjOo3pYE-3cEcZSTbuQ9WCoQfJvL7daofBsBh8CkiHeVtQRx3S9QEUD_edVv5J83uHpyCJaMN3wvIE8K1DbBbhbWEK6WHl46bLHr4Akj-wZzUd8cx10OXBZFq6v5uZiQ73V-GJqP3NhufcXU1p10KbCSwsYiyqNd8F6-4p6Bmg5YucriQ5jM7KXxkTwBb09RNf4B7f2p2_QCw8YpbcwM6IDhdslUHwgRmHwf1fxESvw4im6Vjd6yfVcyjnex9jlItv_dkibvVd5Z-iTDz4_DzM8y1OlXiqRJD55dj0Y6gl5mCDb7RrZIpDC9NHuzdcL0GfetYLEM0hWazBAULPypRsJ79a2RhEvfopoPgkntv10mQmQ1U3X9vo3wRDZoUqfWdiQ2Xy-1kW0Cdg7CM2bSQExmKvdGxj1CVB8fSqFZqBVP4vrXzdQ40hS29rPoEZTg2VqDRK8AKgQjSFr4USaiq-bMq680Ok_y3kaadmEII86o8JtuBeDVKIdYImBsN8QNfIUezoAPgEmFvTGU9fDw4J69CaFWp9anTDCSCKNuRmcs7cZPK7Kz3WVjN35eMTInP2GTDCX-H8tYuzXESKIIrliONkYqGGU4JJ9Lhb11WMg3NJUpTQ3HPFm3tyCgrgMA8--QI0XkLfp50").
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
