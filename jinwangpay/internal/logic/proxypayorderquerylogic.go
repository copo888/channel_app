package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/jinwangpay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strings"
	"time"

	"github.com/copo888/channel_app/jinwangpay/internal/svc"
	"github.com/copo888/channel_app/jinwangpay/internal/types"

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
	data := url.Values{}
	// 組請求參數 FOR JSON
	dataJs := struct {
		MchId         string `json:"mchId"`
		AppId         string `json:"appId"`
		MchOrderNo    string `json:"mchTransOrderNo"`
		ExecuteNotify string `json:"executeNotify"`
		Sign          string `json:"sign"`
	}{
		MchId:         channel.MerId,
		AppId:         "4e09054267a540be91d0b1b04ae116ec",
		MchOrderNo:    req.OrderNo,
		ExecuteNotify: "false",
	}

	sign := payutils.SortAndSignFromObj(dataJs, channel.MerKey, l.ctx)
	dataJs.Sign = sign
	b, err := json.Marshal(dataJs)
	if err != nil {
		fmt.Println("error:", err)
	}
	data.Set("params", string(b))

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%+v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(20).Trace(span).Form(data)

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
		RetCode         string `json:"retCode"`
		RetMsg          string `json:"retMsg, optional"`
		MchId           string `json:"mchId, optional"`
		TransOrderId    string `json:"transOrderId, optional"`
		MchTransOrderNo string `json:"mchTransOrderNo, optional"`
		Amount          string `json:"amount, optional"`
		Currency        string `json:"currency, optional"`
		Status          string `json:"status, optional"` //代付状态::0-订单生成,1-转账中,2-转账成功,3-转账失败
		ChannelOrderNo  string `json:"channelOrderNo, optional"`
		TransSuccTime   int    `json:"transSuccTime, optional"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.RetCode != "SUCCESS" {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s: %s", channelQueryResp.RetCode, channelQueryResp.RetMsg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.RetMsg)
	}
	//代付状态::0-订单生成,1-转账中,2-转账成功,3-转账失败 5-轉帳中。
	var orderStatus = "1"
	if channelQueryResp.Status == "2" {
		orderStatus = "20"
	} else if strings.Index("3", channelQueryResp.Status) > -1 {
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
