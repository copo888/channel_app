package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/miduoduopay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/copo888/channel_app/miduoduopay/internal/svc"
	"github.com/copo888/channel_app/miduoduopay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayQueryBalanceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	traceID string
}

func NewProxyPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayQueryBalanceLogic {
	return ProxyPayQueryBalanceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
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

	timestamp := time.Now().Format("20060102150405")

	data := url.Values{}
	data.Set("MERCHANT_ID", channel.MerId)
	data.Set("VERSION", "1")
	data.Set("SUBMIT_TIME", timestamp)

	//JSON 格式
	//data := struct {
	//	MerchantId   string `json:"merchant_id"`
	//	Version  string `json:"version"`
	//	SubmitTime string `json:"submit_time"`
	//	SignedMsg string `json:"signed_msg"`
	//}{
	//	MerchantId: channel.MerId,
	//	Version: "1",
	//	SubmitTime: timestamp,
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("SIGNED_MSG",sign)
	//data.SignedMsg = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付余额查询请求地址:%s,請求參數:%+v", channel.ProxyPayQueryBalanceUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryBalanceUrl).Timeout(20).Trace(span).Form(data)

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
		Code string `json:"code"`
		Message string `json:"message"`
		Data struct{
			MerchantId string `json:"MERCHANT_ID"`
			MerchantBalanceAvailable string `json:"MERCHANT_BALANCE_AVAILABLE"`
			MerchantDiscount string `json:"MERCHANT_DISCOUNT"`
			MerchantRiskCount string `json:"MERCHANT_RISK_COUNT"`
			MerchantTotalBalance string `json:"MERCHANT_TOTAL_BALANCE"`
		} `json:"data"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if balanceQueryResp.Code != "G_00001" {
		logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %s: %s", balanceQueryResp.Code, balanceQueryResp.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp.Message)
	}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    balanceQueryResp.Data.MerchantBalanceAvailable,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
