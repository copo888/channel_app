package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/jstpay2/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/jstpay2/internal/svc"
	"github.com/copo888/channel_app/jstpay2/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayQueryBalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayQueryBalanceLogic {
	return ProxyPayQueryBalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayQueryBalanceLogic) ProxyPayQueryBalance() (resp *types.ProxyPayQueryInternalBalanceResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayQueryBalance. channelName: %s", l.svcCtx.Config.ProjectName)

	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}
	timestamp := time.Now().Format("20060102150405")
	// 組請求參數 FOR JSON
	data := struct {
		DesTime  string `json:"DesTime"`
		CID      string `json:"CID"`
		BID      string `json:"BID"`
		SignCode string `json:"SignCode"`
	}{
		DesTime: timestamp,
		CID:     channel.MerId,
		BID:     "1",
	}

	// 加簽 JSON
	source := "BID=" + data.BID + "&CID=" + data.CID + "&DesTime=" + data.DesTime + "&Key=" + channel.MerKey
	sign := payutils.GetSign(source)
	logx.Info("加签参数: ", source)
	logx.Info("签名字串: ", sign)
	data.SignCode = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s,請求參數:%#v", channel.ProxyPayQueryUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryBalanceUrl).Timeout(20).Trace(span).JSON(data)

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
		Success bool   `json:"Success"`
		Message string `json:"Message, optional"`
		Result  []struct {
			BalanceAmount float64 `json:"BalanceAmount"`
		} `json:"Result"`
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
		ProxyPayBalance:    fmt.Sprintf("%f", balanceQueryResp.Result[0].BalanceAmount),
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
