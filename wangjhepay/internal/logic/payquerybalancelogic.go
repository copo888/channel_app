package logic

import (
	"context"
	"crypto/tls"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/wangjhepay/internal/payutils"
	"github.com/copo888/channel_app/wangjhepay/internal/svc"
	"github.com/copo888/channel_app/wangjhepay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/url"
	"strconv"
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
	now := time.Now() // current local time
	sec := now.Unix()
	//ip := utils.GetRandomIp()
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	data := url.Values{}
	data.Set("bid", channel.MerId)
	data.Set("time", strconv.FormatInt(sec, 10))
	// 組請求參數 FOR JSON
	//data := struct {
	//	MerchantNumber string `json:"merchantNumber"`
	//	Sign           string `json:"sign"`
	//}{
	//	MerchantNumber: channel.MerId,
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付餘額请求地址:%s,支付餘額請求參數:%#v", channel.PayQueryBalanceUrl, data)
	span := trace.SpanFromContext(l.ctx)
	logx.WithContext(l.ctx).Infof("2")
	// 忽略證書
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	res, ChnErr := gozzle.Post(channel.PayQueryBalanceUrl).Transport(tr).Timeout(20).Trace(span).Form(data)
	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code int64  `json:"code"`
		Msg  string `json:"msg, optional"`
		Time int64  `json:"time"`
		Data struct {
			Money       string `json:"money"`
			LockMoney   string `json:"lock_money"`
			WithdrawMin string `json:"withdraw_min"`
			WithdrawMax string `json:"withdraw_max"`
			WithdrawFee string `json:"withdraw_fee"`
			Wechat      struct {
				Min       int `json:"min"`
				Max       int `json:"max"`
				Switch    int `json:"switch"`
				TimeLimit int `json:"time_limit"`
			} `json:"wechat"`
			Alipay struct {
				Min       int `json:"min"`
				Max       int `json:"max"`
				Switch    int `json:"switch"`
				TimeLimit int `json:"time_limit"`
			} `json:"alipay"`
		} `json:"data"`
	}{}

	if err3 := res.DecodeJSON(&channelResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelResp.Code != 100 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	resp = &types.PayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		WithdrawBalance:    channelResp.Data.Money,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
