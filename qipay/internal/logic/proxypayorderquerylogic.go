package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/qipay/internal/svc"
	"github.com/copo888/channel_app/qipay/internal/types"

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

	url := channel.ProxyPayQueryUrl + "?order_id=" + req.OrderNo
	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%+v", url, req.OrderNo)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Get(url).
		Header("CLIENTSID", channel.MerId).
		Header("ACCESSTOKEN", channel.MerKey).
		Timeout(20).Trace(span).Do()

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
		Code int64  `json:"code"`
		Msg  string `json:"msg, optional"`
		Data []struct {
			OrderSid int64  `json:"order_sid"`
			OrderId  string `json:"order_id"`
			Payer    string `json:"payer"`
			Amount   string `json:"amount"`
			Status   int64  `json:"status"`
			Type     int64  `json:"type"` // 1 為收款,2 為出款
		} `json:"data"`
	}{}

	// 訂單狀態表
	// 0 新訂單
	// 1 已配對
	// 2 已收單
	// 3 已完成
	// 4 回調失敗
	// 90 付款超時
	// 91 收款超時
	// 92 金額不符
	// 95 訂單無效
	// 98 超時配對,無效單
	// 99 超時配對,無效單

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.Code != 200 {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s: %s", channelQueryResp.Code, channelQueryResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Msg)
	} else if len(channelQueryResp.Data) < 1 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, "查无资料")
	}

	channelStatus := channelQueryResp.Data[0].Status
	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelStatus == 3 {
		orderStatus = "20"
	} else if channelStatus >= 90  {
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
