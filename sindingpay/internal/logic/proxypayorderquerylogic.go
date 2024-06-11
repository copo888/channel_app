package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/sindingpay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/sindingpay/internal/svc"
	"github.com/copo888/channel_app/sindingpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderQueryLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewProxyPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderQueryLogic {
	return ProxyPayOrderQueryLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *ProxyPayOrderQueryLogic) ProxyPayOrderQuery(req *types.ProxyPayOrderQueryRequest) (resp *types.ProxyPayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.WithContext(l.ctx).Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	// 組請求參數 FOR JSON
	data := struct {
		MchId   string `json:"mchId"`
		Service string `json:"service"`
		OrderId string `json:"orderId"`
		Sign    string `json:"sign"`
	}{
		MchId:   channel.MerId,
		Service: "Batch.Query",
		OrderId: req.OrderNo,
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%+v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelQueryResp := struct {
		Error  string `json:"Error, optional"`
		Result struct {
			MchId   int    `json:"mchId, optional"`
			OrderId string `json:"orderId, optional"`
			TradeNo int    `json:"tradeNo, optional"`
			Amount  string `json:"amount, optional"`
			Fee     string `json:"fee, optional"`
			Status  int    `json:"status, optional"`
			PayTime string `json:"payTime, optional"`
			Memo    string `json:"memo, optional"`
			Sign    string `json:"sign, optional"`
		} `json:"Result, optional"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.Error != "" {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s", channelQueryResp.Error)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Error)
	}

	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp.Result.Status == 2 { //可选值为0、1、2、-1。0=等待审核，1=处理中，2=成功，-1=失败
		orderStatus = "20"
	} else if channelQueryResp.Result.Status == -1 {
		orderStatus = "30"
	}

	//組返回給BO 的代付返回物件
	return &types.ProxyPayOrderQueryResponse{
		Status: 1,
		//CallBackStatus: ""
		OrderStatus:      orderStatus,
		ChannelReplyDate: time.Now().Format("2006-01-02 15:04:05"),
		//ChannelCharge =
	}, nil
}
