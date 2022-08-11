package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/sandianlingpay/internal/payutils"
	"github.com/copo888/channel_app/sandianlingpay/internal/svc"
	"github.com/copo888/channel_app/sandianlingpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
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
	//timestamp := time.Now().Format("20060102150405")
	//ip := utils.GetRandomIp()
	randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	data := url.Values{}
	data.Set("opmhtid", channel.MerId)
	data.Set("random", randomID)

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
	url := channel.PayQueryBalanceUrl + "?opmhtid=" + channel.MerId + "&random=" + randomID + "&sign=" + sign
	logx.WithContext(l.ctx).Infof("支付餘額请求地址:%s,支付餘額請求參數:%#v", url, data)
	span := trace.SpanFromContext(l.ctx)
	//res, ChnErr := gozzle.Post(channel.PayQueryBalanceUrl).Timeout(20).Trace(span).JSON(data)

	res, ChnErr := gozzle.Get(url).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		ErrorCode int    `json:"errorCode"`
		ErrorMsg  string `json:"errorMsg"`
		Result    struct {
			Balanceavailable int `json:"balanceavailable"`
			Balancereal      int `json:"balancereal"`
		}
	}{}

	if err3 := res.DecodeJSON(&channelResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}
	if channelResp.ErrorCode != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg)
	}

	balanceStr := strconv.Itoa(channelResp.Result.Balancereal)
	balance := utils.FloatDiv(balanceStr, "100")
	b := fmt.Sprintf("%v", balance)

	resp = &types.PayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		WithdrawBalance:    b,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
