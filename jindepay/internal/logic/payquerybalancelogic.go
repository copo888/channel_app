package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/jindepay/internal/payutils"
	"github.com/copo888/channel_app/jindepay/internal/svc"
	"github.com/copo888/channel_app/jindepay/internal/types"
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

	// 組請求參數 FOR JSON
	data := struct {
		MerchantId string `json:"merchant_id"`
		Timestamp  string `json:"timestamp"`
		Sign       string `json:"sign"`
		SignType   string `json:"sign_type"`
	}{
		MerchantId: channel.MerId,
		Timestamp:  string(time.Now().Unix()),
		Sign:       "",
		SignType:   "MD5",
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付餘額请求地址:%s,支付餘額請求參數:%#v", channel.PayQueryBalanceUrl, data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayQueryBalanceUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}

	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理
	channelRespMsg := struct {
		Code int64  `json:"code"`
		Msg  string `json:"message"`
	}{}

	channelResp := struct {
		Code int64  `json:"code"`
		Msg  string `json:"message"`
		Data struct {
			Money float64 `json:"money"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelRespMsg); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelRespMsg.Code != 200 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelRespMsg.Msg)
	}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	resp = &types.PayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		WithdrawBalance:    fmt.Sprintf("%f", channelResp.Data.Money),
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
