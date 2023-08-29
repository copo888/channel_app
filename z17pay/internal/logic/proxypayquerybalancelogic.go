package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/z17pay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/copo888/channel_app/z17pay/internal/svc"
	"github.com/copo888/channel_app/z17pay/internal/types"

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

	data := url.Values{}
	data.Set("account", channel.MerId)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("token", sign)

	url := channel.ProxyPayQueryBalanceUrl + "?account=" + channel.MerId + "&token=" + sign
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
		Data struct {
			StoreAccount          string `json:"store_account, optional"`
			StoreMoney            string `json:"store_money, optional"`
			StoreRealpaidmoney    string `json:"store_realpaidmoney, optional"`
			StoreRealreceivemoney string `json:"store_realreceivemoney, optional"`
			StorePayMin           string `json:"store_pay_min, optional"`
			StorePayMax           string `json:"store_pay_max, optional"`
			StoreReceiveMin       string `json:"store_receive_min, optional"`
			StoreReceiveMax       string `json:"store_receive_max, optional"`
			StoreWithdrawMin      string `json:"store_withdraw_min, optional"`
			StoreWithdrawMax      string `json:"store_withdraw_max, optional"`
			StoreRechargeMin      string `json:"store_recharge_min, optional"`
			StoreRechargeMax      string `json:"store_recharge_max, optional"`
			CToCPaybase           string `json:"c_to_c_paybase, optional"`
			CToCReceiverate       string `json:"c_to_c_receiverate, optional"`
			AToCReceiverate       string `json:"a_to_c_receiverate, optional"`
		} `json:"data, optional"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&balanceQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}
	//else if balanceQueryResp.Success != true {
	//	logx.WithContext(l.ctx).Errorf("代付余额查询渠道返回错误: %s: %s", balanceQueryResp.Success, balanceQueryResp.Msg)
	//	return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, balanceQueryResp.Msg)
	//}

	resp = &types.ProxyPayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		ProxyPayBalance:    balanceQueryResp.Data.StoreMoney,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
