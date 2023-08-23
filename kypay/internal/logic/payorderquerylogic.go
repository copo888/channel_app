package logic

import (
	"context"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/kypay/internal/payutils"
	"github.com/copo888/channel_app/kypay/internal/svc"
	"github.com/copo888/channel_app/kypay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
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
		MerchId      string `json:"merchant_serial"`
		OrderId      string `json:"merchant_order_no"`
		RandomString string `json:"random_string"`
		Sign         string `json:"sign"`
	}{
		MerchId:      channel.MerId,
		OrderId:      req.OrderNo,
		RandomString: payutils.GetSign(req.OrderNo),
	}

	// 加簽 JSON
	sign := payutils.SortAndSignSHA256FromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

		channelResp2 := struct {
			Msg string `json:"message,optional"`
		}{}

		// 返回body 轉 struct
		if err = res.DecodeJSON(&channelResp2); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp2.Msg)
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		MerchantSerial  string      `json:"merchant_serial"`
		MerchantOrderNo string      `json:"merchant_order_no"`
		PlatformOrderNo string      `json:"platform_order_no"`
		RandomString    string      `json:"random_string"`
		Amount          string      `json:"amount"`
		VerifyAmount    string      `json:"verify_amount"`
		RealAmount      string      `json:"real_amount"`
		SucceededAt     interface{} `json:"succeeded_at"`
		Status          string      `json:"status"`
		Sign            string      `json:"sign"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}
	//else if channelResp.Status != "success" {
	//	return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Status)
	//}
	//unpaid 未⽀付
	//paid 已⽀付
	//completed 已完成

	orderStatus := "0"
	if channelResp.Status == "paid" || channelResp.Status == "completed" {
		orderStatus = "1"
	}

	amount, _ := strconv.ParseFloat(channelResp.RealAmount, 64)

	resp = &types.PayOrderQueryResponse{
		OrderAmount: amount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
