package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/wtpay/internal/svc"
	"github.com/copo888/channel_app/wtpay/internal/types"

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
	auth := "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VySWQiOjAsInBsYXRmb3JtSWQiOjE1OCwiYWdlbnRJZCI6MCwidmVyc2lvbiI6MSwicGF5bWVudElkIjowLCJpYXQiOjE2NTM0NzUwNjd9.N1FBN6L95D4n1UxBtuoC464gbeCZsb5RKQunWhwWPew"

	logx.WithContext(l.ctx).Infof("Enter ProxyPayQueryBalance. channelName: %s", l.svcCtx.Config.ProjectName)

	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%#v", channel.ProxyPayQueryBalanceUrl)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Get(channel.ProxyPayQueryBalanceUrl).Header("Authorization", auth).Timeout(20).Trace(span).Do()

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
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			TotalBalance float64 `json:"total_balance"`
		} `json:"data"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, err3.Error())
	} else if balanceQueryResp.ErrorCode != "0000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp.ErrorMsg)
	}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    fmt.Sprintf("%f", utils.FloatDivF(balanceQueryResp.Data.TotalBalance, 100)),
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
