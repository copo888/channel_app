package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/shenfutongpay7/internal/payutils"
	"github.com/copo888/channel_app/shenfutongpay7/internal/svc"
	"github.com/copo888/channel_app/shenfutongpay7/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"time"
)

type PayQueryBalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayQueryBalanceLogic {
	return PayQueryBalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayQueryBalanceLogic) PayQueryBalance() (resp *types.PayQueryInternalBalanceResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayQueryBalance. channelName: %s", l.svcCtx.Config.ProjectName)

	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	// 取值
	//timestamp := time.Now().Format("20060102150405")
	//ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	//data := url.Values{}
	//data.Set("merchant_number", channel.MerId)

	// 組請求參數 FOR JSON
	data := struct {
		Nonce string `json:"nonce"`
	}{
		Nonce: randomID,
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)
	b, errM := json.Marshal(data)
	if errM != nil {
		return nil, errorx.New(responsex.SYSTEM_ERROR, "JSON解析失败")
	}
	dataJsonStr := string(b) + channel.MerKey
	sign := payutils.GetSign_SHA1(dataJsonStr)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付餘額请求地址:%s,支付餘額請求參數:%+v", channel.PayQueryBalanceUrl, data)
	logx.WithContext(l.ctx).Infof("加签参数: %+v, Sign: %s", dataJsonStr, sign)
	span := trace.SpanFromContext(l.ctx)
	headers := make(map[string]string)
	headers["X-Imx-Mer"] = channel.MerId
	headers["X-Imx-Sign"] = sign
	res, ChnErr := gozzle.Post(channel.PayQueryBalanceUrl).Headers(headers).Timeout(20).Trace(span).JSON(data)
	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code string `json:"code"`
		Message string `json:"message"`
		Bals struct{
			Cny struct {
				Bal float64 `json:"bal"`
				Pending float64 `json:"pending"`
				Frozen float64 `json:"frozen"`
			}
		} `json:"bals"`
	}{}

	if err3 := res.DecodeJSON(&channelResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}

	resp = &types.PayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		WithdrawBalance:    fmt.Sprintf("%f",channelResp.Bals.Cny.Bal),
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
