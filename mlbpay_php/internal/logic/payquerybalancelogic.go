package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/mlbpay_php/internal/payutils"
	"github.com/copo888/channel_app/mlbpay_php/internal/svc"
	"github.com/copo888/channel_app/mlbpay_php/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"time"
)

type PayQueryBalanceLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayQueryBalanceLogic {
	return PayQueryBalanceLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
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
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	// 組請求參數
	//data := url.Values{}
	//data.Set("merchant_number", channel.MerId)

	// 組請求參數 FOR JSON
	reqData := struct {
		Sign        string `json:"sign"`
		Context     []byte `json:"context"`
		EncryptType string `json:"encryptType"`
	}{}

	data := struct {
		MerchNo string `json:"merchNo"`
	}{
		MerchNo: channel.MerId,
	}

	out, errJ := json.Marshal(data)
	if errJ != nil {
		return nil, errJ
	}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	//data.Set("sign", sign)
	//source := strings.ToValidUTF8(string(jj), "")+channel.MerKey
	source := string(out) + channel.MerKey
	sign := payutils.GetSign(source)
	logx.WithContext(l.ctx).Info("sign加签参数: ", source)
	logx.WithContext(l.ctx).Info("context加签参数: ", string(out))
	reqData.Sign = sign
	reqData.Context = out
	reqData.EncryptType = "MD5"

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付餘額请求地址:%s,支付餘額請求參數:%+v", channel.PayQueryBalanceUrl, reqData)
	logx.WithContext(l.ctx).Infof("支付餘額加密請求參數:%+v", data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayQueryBalanceUrl).Timeout(20).Trace(span).JSON(reqData)
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
		Code int    `json:"code"`
		Msg  string `json:"msg, optional"`
	}{}

	if err3 := res.DecodeJSON(&channelResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}

	// 渠道狀態碼判斷
	if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	channelResp2 := struct {
		Sign    string `json:"sign,optional"`
		Context []byte `json:"context,optional"`
	}{}

	if err3 := res.DecodeJSON(&channelResp2); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}

	respCon := struct {
		MerchNo  string `json:"merchNo"`
		AvailBal string `json:"availBal"`
	}{}

	json.Unmarshal(channelResp2.Context, &respCon)
	logx.WithContext(l.ctx).Errorf("支付馀额查询渠道返回参数解密: %s", respCon)
	resp = &types.PayQueryInternalBalanceResponse{
		ChannelNametring:   channel.Name,
		ChannelCodingtring: channel.Code,
		WithdrawBalance:    respCon.AvailBal,
		UpdateTimetring:    time.Now().Format("2006-01-02 15:04:05"),
	}

	return
}
