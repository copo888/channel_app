package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/senlinpay2/internal/payutils"
	"github.com/copo888/channel_app/senlinpay2/internal/svc"
	"github.com/copo888/channel_app/senlinpay2/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"time"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %#v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	timestamp := time.Now().Format("20060102150405")
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	// 組請求參數
	//if req.OrderNo != "" {
	//	data.Set("trade_no", req.OrderNo)
	//}
	//if req.ChannelOrderNo != "" {
	//	data.Set("order_no", req.ChannelOrderNo)
	//}
	//data.Set("appid", channel.MerId)
	//data.Set("nonce_str", randomID)

	// 組請求參數 FOR JSON
	data := url.Values{}
	data.Add("mchId", channel.MerId)    // 使用 channel.MerId 变量
	data.Add("mchOrderNo", req.OrderNo) // 使用 req.OrderNo 变量
	data.Add("reqTime", timestamp)      // 使用 req.OrderNo 变量
	data.Add("version", "1.0")          // 使用 req.OrderNo 变量
	data.Add("executeNotify", "false")  // 直接使用字符串字面量 "false"

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("sign", sign)

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		RetCode        string `json:"retCode"`
		Sign           string `json:"sign"`
		MchId          string `json:"mchId"`
		AppId          string `json:"appId"`
		ProductId      string `json:"productId"`
		PayOrderId     string `json:"payOrderId"`
		MchOrderNo     string `json:"mchOrderNo"`
		Amount         string `json:"amount"`
		Currency       string `json:"currency"`
		Status         string `json:"status"` //当前订单状态: -2:订单	已关闭,0	订单⽣成,1-⽀付中,2-⽀付成功,3-业务处理完成,4-已退款（2和3都表示⽀付成功,3表示⽀付平台回调商户且返回成功后的状态）
		ChannelUser    string `json:"channelUser"`
		ChannelOrderNo string `json:"channelOrderNo"`
		ChannelAttach  string `json:"channelAttach"`
		PaySuccTime    string `json:"paySuccTime"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.RetCode != "0" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR)
	}

	amountF, _ := strconv.ParseFloat(channelResp.Amount, 64)
	orderAmount := utils.FloatDivF(amountF, 100) // 單位:元

	orderStatus := "0"
	if channelResp.Status == "2" || channelResp.Status == "3" { //当前订单状态: -2:订单已关闭,0	订单⽣成,1-⽀付中,2-⽀付成功,3-业务处理完成,4-已退款（2和3都表示⽀付成功,3表示⽀付平台回调商户且返回成功后的状态）
		orderStatus = "1"
	} else if channelResp.Status == "4" {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
