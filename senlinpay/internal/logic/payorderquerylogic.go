package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/senlinpay/internal/payutils"
	"github.com/copo888/channel_app/senlinpay/internal/svc"
	"github.com/copo888/channel_app/senlinpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayOrderQueryLogic) PayOrderQuery(req *types.PayOrderQueryRequest) (resp *types.PayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	// 組請求參數 FOR JSON
	data := struct {
		MchId      string `json:"mchId"`
		OutTradeNo string `json:"outTradeNo"`
		ReqTime    int64  `json:"reqTime"`
		Sign       string `json:"sign"`
	}{
		MchId:      channel.MerId,
		OutTradeNo: req.OrderNo,
		ReqTime:    timestamp,
	}

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		Code    int    `json:"code, optional"`
		Message string `json:"message,optional"`
		Data    struct {
			MchId         string `json:"mchId, optional"`
			WayCode       int    `json:"wayCode, optional"`
			TradeNo       string `json:"tradeNo, optional"`
			OutTradeNo    string `json:"outTradeNo, optional"`
			OriginTradeNo string `json:"originTradeNo, optional"`
			Amount        string `json:"amount, optional"`
			Subject       string `json:"subject, optional"`
			Body          string `json:"body, optional"`
			ExtParam      string `json:"extParam, optional"`
			NotifyUrl     string `json:"notifyUrl, optional"`
			PayUrl        string `json:"payUrl, optional"`
			ExpiredTime   string `json:"expiredTime, optional"`
			SuccessTime   string `json:"successTime, optional"`
			CreateTime    string `json:"createTime, optional"`
			State         int    `json:"state, optional"`       //订单状态：0=待支付，1=支付成功，2=支付失败，3=未出码，4=异常
			NotifyState   int    `json:"notifyState, optional"` //通知状态：0=未通知，1=通知成功，2=通知失败
		} `json:"data, optional"`
		Sign string `json:"sign, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Message)
	}

	orderAmount := utils.FloatDiv(channelResp.Data.Amount, "100")

	orderStatus := "0"
	if channelResp.Data.State == 1 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
