package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/shahepay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/copo888/channel_app/shahepay/internal/svc"
	"github.com/copo888/channel_app/shahepay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderQueryLogic {
	return ProxyPayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayOrderQueryLogic) ProxyPayOrderQuery(req *types.ProxyPayOrderQueryRequest) (resp *types.ProxyPayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.WithContext(l.ctx).Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	timestamp := time.Now().Format(time.RFC3339)
	// 組請求參數 FOR JSON
	params := struct {
		AccountName     string `json:"account_name"`
		MerchantOrderId string `json:"merchant_order_id"`
		Timestamp       string `json:"timestamp"`
	}{
		AccountName:     channel.MerId,
		MerchantOrderId: req.OrderNo,
		Timestamp:       timestamp,
	}

	// 加簽
	paramsJson, _ := json.Marshal(params)
	signature := payutils.GetSign2(paramsJson, l.svcCtx.PrivateKey)
	// 組請求參數
	data := url.Values{}
	data.Set("data", string(paramsJson))
	data.Set("signature", signature)

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%#v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(20).Trace(span).Form(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, chnErr.Error())
	} else if res.Status() == 403 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, fmt.Sprintf("Error HTTP Status: %d, %s", res.Status(), string(res.Body())))
	} else if res.Status() < 200 && res.Status() >= 300 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	// 渠道回覆處理
	channelResp := struct {
		ID          string `json:"id"`
		TotalAmount string `json:"total_amount"`
		Status      string `json:"status"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelResp.Status == "completed" {
		orderStatus = "20"
	} else if channelResp.Status == "failed" {
		orderStatus = "30"
	}

	//組返回給BO 的代付返回物件
	return &types.ProxyPayOrderQueryResponse{
		Status: 1,
		//CallBackStatus: ""
		ChannelOrderNo:   channelResp.ID,
		OrderStatus:      orderStatus,
		ChannelReplyDate: time.Now().Format("2006-01-02 15:04:05"),
		//ChannelCharge =
	}, nil
}
