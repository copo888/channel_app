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
	logx.WithContext(l.ctx).Infof("AccessToken: %s", l.svcCtx.Config.AccessToken)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).
		Header("Authorization", "Bearer "+l.svcCtx.Config.AccessToken).
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
