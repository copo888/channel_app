package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/htpay2/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/copo888/channel_app/htpay2/internal/svc"
	"github.com/copo888/channel_app/htpay2/internal/types"

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

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.WithContext(l.ctx).Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)
	randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	//data := struct {
	//	MerchId      string `json:"mch_id"`
	//	DfMchOrderNo string `json:"df_mch_order_no"`
	//	Sign         string `json:"sign"`
	//}{
	//	MerchId:      channel.MerId,
	//	DfMchOrderNo: req.OrderNo,
	//}
	data := url.Values{}
	data.Set("mhtorderno", req.OrderNo)
	data.Set("opmhtid", channel.MerId)
	data.Set("random", randomID)
	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)

	queryUrl := channel.ProxyPayQueryUrl + "?mhtorderno=" + req.OrderNo + "&opmhtid=" + channel.MerId + "&random=" + randomID + "&sign=" + sign

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%#v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Get(queryUrl).Timeout(20).Trace(span).Do()

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
		RtCode int    `json:"rtCode"`
		Msg    string `json:"msg"`
		Result struct {
			Pforderno      string  `json:"pforderno,optional"`
			Orderamount    float64 `json:"orderamount,optional"`
			Paidamount     float64 `json:"paidamount,optional"`
			Currency       string  `json:"currency,optional"`
			Acctype        string  `json:"acctype,optional"`
			Bankcode       string  `json:"bankcode,optional"`
			Accprovince    string  `json:"accprovince,optional"`
			Acccityname    string  `json:"acccityname,optional"`
			Accname        string  `json:"accname,optional"`
			Accno          string  `json:"accno,optional"`
			Remark         string  `json:"remark,optional"`
			Ordertime      string  `json:"ordertime,optional"`
			Status         int     `json:"status,optional"`
			Settletime     string  `json:"settletime,optional"`
			Notifyurl      string  `json:"notifyurl,optional"`
			Notifystatus   int     `json:"notifystatus,optional"`
			Lastnotifytime string  `json:"lastnotifytime,optional"`
			Beforebalance  float64 `json:"beforebalance,optional"`
			Afterbalance   float64 `json:"afterbalance,optional"`
			Fromip         string  `json:"fromip,optional"`
		} `json:"result,optional"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.RtCode != 0 {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.Msg)
	}
	//0：处理中	//1：成功	//2：失败
	var orderStatus = "1"
	if channelQueryResp.Result.Status == 1 {
		orderStatus = "20"
	} else if channelQueryResp.Result.Status == 2 {
		orderStatus = "30"
	}

	//組返回給BO 的代付返回物件
	return &types.ProxyPayOrderQueryResponse{
		Status: 1,
		//CallBackStatus: ""
		ChannelOrderNo:   channelQueryResp.Result.Pforderno,
		OrderStatus:      orderStatus,
		ChannelReplyDate: time.Now().Format("2006-01-02 15:04:05"),
		//ChannelCharge =
	}, nil
}
