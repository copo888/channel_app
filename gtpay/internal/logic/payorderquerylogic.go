package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/gtpay/internal/payutils"
	"github.com/copo888/channel_app/gtpay/internal/svc"
	"github.com/copo888/channel_app/gtpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
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

	// 組請求參數 FOR JSON
	data := struct {
		MerchantId string `json:"merchant_id"`
		OutTradeNo string `json:"out_trade_no"`
		Sign       string `json:"sign"`
		SignType   string `json:"sign_type"`
	}{
		MerchantId: channel.MerId,
		OutTradeNo: req.OrderNo,
		SignType:   "md5",
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
	channelResp1 := struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}{}

	channelResp := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MerchantId int     `json:"merchant_id"`
			OutTradeNo string  `json:"out_trade_no"`
			TradeNo    string  `json:"trade_no"`
			Money      float64 `json:"money"`
			MoneyTrue  float64 `json:"money_true"`
			Fee        float64 `json:"fee"`
			State      int     `json:"state"`
			Sign       string  `json:"sign"`
			SignType   string  `json:"sign_type"`
		} `json:"data"`
	}{}

	if err = res.DecodeJSON(&channelResp1); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp1.Code != 200 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp1.Message)
	} else if channelResp1.Code == 200 {
		if err = res.DecodeJSON(&channelResp); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}
	}

	//orderAmount, errParse := strconv.ParseFloat(channelResp.Data.MoneyTrue, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.Data.State == 1 { //1 成功，0 待支付，-1 失败
		orderStatus = "1"
	}

	var amount float64
	if channelResp.Data.MoneyTrue != 0 {
		amount = channelResp.Data.MoneyTrue
	} else {
		amount = channelResp.Data.Money
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: amount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
