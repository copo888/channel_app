package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/wtpay/internal/svc"
	"github.com/copo888/channel_app/wtpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderQueryLogic) PayOrderQuery(req *types.PayOrderQueryRequest) (resp *types.PayOrderQueryResponse, err error) {
	auth := "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VySWQiOjAsInBsYXRmb3JtSWQiOjE1OCwiYWdlbnRJZCI6MCwidmVyc2lvbiI6MSwicGF5bWVudElkIjowLCJpYXQiOjE2NTM0NzUwNjd9.N1FBN6L95D4n1UxBtuoC464gbeCZsb5RKQunWhwWPew"

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	// 組請求參數
	data := url.Values{}
	data.Set("payment_cl_id", req.OrderNo)
	data.Set("Authorization", auth)
	payQueryUrl := channel.PayQueryUrl + "?payment_cl_id=" + req.OrderNo
	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", payQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Get(payQueryUrl).Header("Authorization", auth).Timeout(20).Trace(span).Do()

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	type PayData struct {
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      []struct {
			Amount      float64 `json:"amount"` // 單位:分
			RealAmount  float64 `json:"real_amount"`
			PaymentId   string  `json:"payment_id"`
			PaymentClId string  `json:"payment_cl_id"`
			Status      int64   `json:"status"`
		} `json:"data"`
	}
	// 渠道回覆處理
	var channelResp PayData

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.ErrorCode != "0000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg)
	} else if len(channelResp.Data) == 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, "查無資料")
	}

	orderStatus := "0"
	if channelResp.Data[0].Status == 2 {
		orderStatus = "1"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: utils.FloatDivF(channelResp.Data[0].Amount, 100), // 單位:元
		OrderStatus: orderStatus,                                      //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
