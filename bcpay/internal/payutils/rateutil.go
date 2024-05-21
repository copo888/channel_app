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
		FromCurrency string `json:"from_currency"`
		ToCurrency   string `json:"to_currency"`
		Type         string `json:"type"`
	}{
		Command:      "rates",
		Amount:       "100", //要转换加密货币的"法币数额" 预设100，换算要先除100再换算
		FromCurrency: exchangeInfo.Currency,
		ToCurrency:   exchangeInfo.Token,
		Type:         "BUY",
	}

	res, ChnErr := gozzle.Post(exchangeInfo.Url).Timeout(20).Trace(span).
		Header("Authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6IjRiNjY2YjJiMjU4OTk2NjYyYjdjMzMzOWNlOTQ2OGI0ZTFmMGJmOWFlM2U0MTk2YjM4YThjNGE5ZGIzODZmNTMyZjkxMTk5YmExNTMwZDJlIn0.eyJhdWQiOiIxIiwianRpIjoiNGI2NjZiMmIyNTg5OTY2NjJiN2MzMzM5Y2U5NDY4YjRlMWYwYmY5YWUzZTQxOTZiMzhhOGM0YTlkYjM4NmY1MzJmOTExOTliYTE1MzBkMmUiLCJpYXQiOjE3MTUwNTMyNDYsIm5iZiI6MTcxNTA1MzI0NiwiZXhwIjoxNzE1NjU4MDQ1LCJzdWIiOiI0NDkiLCJzY29wZXMiOltdfQ.F5kXd0iUAxMG5EU9D33gdIGIe58r5OHDfun-xfXZ0L7hoIdWZXsudL9kR637r4b_MRQz8oeUeOuAFFwF0eEHxW-0YtE6tySzJggwwHE2TRnjrleG3WlQUpIudiu_J9QCU03mJMWGqJyyAeRLL0julZYX5U3zpk0Bl5gzOH7BgQgcBRCUq8mKyR-QtO6IJLP6HLlSaRVNoM1_Ze8C7VgX9Fyko95ALTENrlr8DWggGkqoimK8vMmkxcMs06B8f3tIBY0XyMi9WnVaCVhMxjrMFik9DsVAr9QOXcKoxo-tO3k8-5oG75jmRLitVzt4vtLfbSnPShP2cmJPMSj6xSoIoosMW3mg0zPk8N--SaOy2uBf-Qhle3kBg44OJSY0q_7f33WYjgLp-8vpPoaCML2Q_Hd85iza0Yn1EwM1axGfXnDAX80w-y-6wSjrdVCGPO3XyV3tb8wGfSc_Ga5F7UFsKVZTm-Il4_DqPQXIXcCZtKk-i2qQ4Ksdaq_uuf4ZdOUHLiWth3zpvzGRw2n2A5gvRtESfHAS454ntt61c5aCLxkUhy04XYvhZtPsv1vSCOEcXxnmMGc11_wGQeZHodYdTRSBkSay_-jav3yaWzqswpZ3Q5BzFoZKHDFkcRftwICz7624T7fiC5iLnYIL6y8oqf-WMWoLf3JQ71b_5BR9eBU").
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
