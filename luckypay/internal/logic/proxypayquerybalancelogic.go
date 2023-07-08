package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/luckypay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"time"

	"github.com/copo888/channel_app/luckypay/internal/svc"
	"github.com/copo888/channel_app/luckypay/internal/types"

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

	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	data := url.Values{}
	data.Set("clientCode", channel.MerId)
	data.Set("coinChain", "BANK")
	data.Set("coinUnit", "THB")
	data.Set("requestTimestamp", fmt.Sprintf("%d", timestamp))

	// 加簽
	keys := []string{"clientCode", "coinChain", "coinUnit", "requestTimestamp"}
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, keys, l.ctx)
	data.Set("sign", sign)

	url := channel.ProxyPayQueryBalanceUrl + "?clientCode=" + channel.MerId + "&coinChain=BANK&coinUnit=THB&requestTimestamp=" + fmt.Sprintf("%d", timestamp) + "&sign=" + sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s,請求參數:%+v", url, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Get(url).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	balanceQueryResp := struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message,optional"`
		Data    struct {
			Sign             string `json:"sign,optional"`
			CheckRequestTime int64  `json:"checkRequestTime,optional"`
			DfBalance        struct {
				CoinChain string  `json:"coinChain,optional"`
				CoinUnit  string  `json:"coinUnit,optional"`
				Balance   float64 `json:"balance,optional"`
				Freeze    float64 `json:"freeze,optional"`
			} `json:"dfBalance,optional"`
			DsBalance struct {
				CoinChain string  `json:"coinChain,optional"`
				CoinUnit  string  `json:"coinUnit,optional"`
				Balance   float64 `json:"balance,optional"`
				Freeze    float64 `json:"freeze,optional"`
			} `json:"dsBalance,optional"`
		} `json:"data,optional"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if balanceQueryResp.Success != true {
		logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %s: %s", balanceQueryResp.Success, balanceQueryResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp.Message)
	}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    strconv.FormatFloat(balanceQueryResp.Data.DfBalance.Balance, 'f', 2, 64),
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
