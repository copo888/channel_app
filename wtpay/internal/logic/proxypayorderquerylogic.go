package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/wtpay/internal/svc"
	"github.com/copo888/channel_app/wtpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"

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
	auth := "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VySWQiOjAsInBsYXRmb3JtSWQiOjE1OCwiYWdlbnRJZCI6MCwidmVyc2lvbiI6MSwicGF5bWVudElkIjowLCJpYXQiOjE2NTM0NzUwNjd9.N1FBN6L95D4n1UxBtuoC464gbeCZsb5RKQunWhwWPew"

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.WithContext(l.ctx).Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}
	// 組請求參數
	data := url.Values{}
	data.Set("payment_cl_id", req.OrderNo)
	data.Set("Authorization", auth)
	proxyPayQueryUrl := channel.ProxyPayQueryUrl + "?payout_cl_id=" + req.OrderNo

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%#v", proxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Get(proxyPayQueryUrl).Header("Authorization", auth).Timeout(20).Trace(span).Do()

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理
	channelQueryResp := struct {
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      []struct {
			Amount     float64 `json:"amount"` // 單位:分
			PayoutId   string  `json:"payout_id"`
			PayoutClId string  `json:"payout_cl_id"`
			Status     int64   `json:"status"`
		} `json:"data"`
	}{}

	if err = ChannelResp.DecodeJSON(&channelQueryResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelQueryResp.ErrorCode != "0000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.ErrorMsg)
	} else if len(channelQueryResp.Data) == 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, "查無資料")
	}

	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp.Data[0].Status == 3 {
		orderStatus = "20"
	} else if channelQueryResp.Data[0].Status == 4 || channelQueryResp.Data[0].Status == 5 {
		orderStatus = "30"
	}

	//組返回給BO 的代付返回物件
	resp = &types.ProxyPayOrderQueryResponse{
		Status:           1,
		ChannelOrderNo:   channelQueryResp.Data[0].PayoutId,
		OrderStatus:      orderStatus,
		ChannelReplyDate: "",
		ChannelCharge:    0,
	}

	return resp, nil
}
