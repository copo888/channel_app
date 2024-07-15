package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/bcpay/internal/payutils"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/bcpay/internal/svc"
	"github.com/copo888/channel_app/bcpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayQueryBalanceLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewProxyPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayQueryBalanceLogic {
	return ProxyPayQueryBalanceLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *ProxyPayQueryBalanceLogic) ProxyPayQueryBalance() (resp *types.ProxyPayQueryInternalBalanceResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayQueryBalance. channelName: %s", l.svcCtx.Config.ProjectName)

	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	// 組請求參數 FOR JSON
	data := struct {
		Command  string `json:"command"`
		HashCode string `json:"hashCode"`
	}{
		Command:  "balances",
		HashCode: payutils.GetSign("balances" + channel.MerKey),
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s,請求參數:%+v", channel.ProxyPayQueryBalanceUrl, data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).
		Header("Authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImp0aSI6IjYxYWRhNzM3ZWNmMzIwMjE3ZmVlYzUyZDIzNDgyNTkwOTI1YjAyNjI2YzY1MjAwMDk4ODc1ZmY2NzI2N2FkMjdkNGY3MmQ5NmVkYzgzNDY4In0.eyJhdWQiOiIxIiwianRpIjoiNjFhZGE3MzdlY2YzMjAyMTdmZWVjNTJkMjM0ODI1OTA5MjViMDI2MjZjNjUyMDAwOTg4NzVmZjY3MjY3YWQyN2Q0ZjcyZDk2ZWRjODM0NjgiLCJpYXQiOjE3MjA2ODcxMjQsIm5iZiI6MTcyMDY4NzEyNCwiZXhwIjoxNzIxMjkxOTI0LCJzdWIiOiI0NDkiLCJzY29wZXMiOltdfQ.loWg5juLro4FtuwOY-5ui_D1CQ_IcCZjOo3pYE-3cEcZSTbuQ9WCoQfJvL7daofBsBh8CkiHeVtQRx3S9QEUD_edVv5J83uHpyCJaMN3wvIE8K1DbBbhbWEK6WHl46bLHr4Akj-wZzUd8cx10OXBZFq6v5uZiQ73V-GJqP3NhufcXU1p10KbCSwsYiyqNd8F6-4p6Bmg5YucriQ5jM7KXxkTwBb09RNf4B7f2p2_QCw8YpbcwM6IDhdslUHwgRmHwf1fxESvw4im6Vjd6yfVcyjnex9jlItv_dkibvVd5Z-iTDz4_DzM8y1OlXiqRJD55dj0Y6gl5mCDb7RrZIpDC9NHuzdcL0GfetYLEM0hWazBAULPypRsJ79a2RhEvfopoPgkntv10mQmQ1U3X9vo3wRDZoUqfWdiQ2Xy-1kW0Cdg7CM2bSQExmKvdGxj1CVB8fSqFZqBVP4vrXzdQ40hS29rPoEZTg2VqDRK8AKgQjSFr4USaiq-bMq680Ok_y3kaadmEII86o8JtuBeDVKIdYImBsN8QNfIUezoAPgEmFvTGU9fDw4J69CaFWp9anTDCSCKNuRmcs7cZPK7Kz3WVjN35eMTInP2GTDCX-H8tYuzXESKIIrliONkYqGGU4JJ9Lhb11WMg3NJUpTQ3HPFm3tyCgrgMA8--QI0XkLfp50").
		Header("Content-type", "application/json").
		JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	balanceQueryResp := struct {
		Crypto struct {
			ERC20USDT   string `json:"ERC20-USDT"`
			ERC20USDC   string `json:"ERC20-USDC"`
			ETH         string `json:"ETH"`
			TRC20USDT   string `json:"TRC20-USDT"`
			TRC20USDC   string `json:"TRC20-USDC"`
			TRX         string `json:"TRX"`
			BTC         string `json:"BTC"`
			BSCBNB      string `json:"BSC-BNB"`
			BEP20DOGE   string `json:"BEP20-DOGE"`
			BEP20EOS    string `json:"BEP20-EOS"`
			BEP20BUSD   string `json:"BEP20-BUSD"`
			ERC20DAI    string `json:"ERC20-DAI"`
			ERC20SHIB   string `json:"ERC20-SHIB"`
			ERC20BUSD   string `json:"ERC20-BUSD"`
			XRP         string `json:"XRP"`
			ADA         string `json:"ADA"`
			NETWORKUSDT string `json:"NETWORK-USDT"`
		} `json:"crypto"`
		Fiat struct {
			MYR string `json:"MYR"`
			THB string `json:"THB"`
			IDR string `json:"IDR"`
			VND string `json:"VND"`
			KRW string `json:"KRW"`
			BRL string `json:"BRL"`
			BDT string `json:"BDT"`
			INR string `json:"INR"`
		} `json:"fiat"`
	}{}

	if err3 := res.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if res.Status() >= 300 || res.Status() < 200 {
		logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %+v", balanceQueryResp)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, fmt.Sprintf("%+v", balanceQueryResp))
	}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    balanceQueryResp.Crypto.BTC,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
