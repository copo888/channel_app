package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/yuehhaopay/internal/payutils"
	"github.com/copo888/channel_app/yuehhaopay/internal/svc"
	"github.com/copo888/channel_app/yuehhaopay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
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
	timestamp := time.Now().Unix()
	timeStr := strconv.FormatInt(timestamp, 10)
	// 組請求參數
	data := url.Values{}
	data.Set("orderid", req.OrderNo)
	data.Set("uid", channel.MerId)
	data.Set("timestamp", timeStr)

	// 組請求參數 FOR JSON

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("sign", sign)

	// 加簽 JSON
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

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
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code   int    `json:"status"`
		Sign   string `json:"sign"`
		Msg    string `json:"msg, optional"`
		Result struct {
			Data struct {
				Zero struct {
					TransactionId int64  `json:"transactionid, optional"`
					OrderId       string `json:"orderid, optional"`
					Channel       int    `json:"channel, optional"`
					Amount        string `json:"amount, optional"`
					RealAmount    string `json:"real_amount, optional"`
					Status        int    `json:"status, optional"` //0:未处理 1:交易成功 	2:处理中 3:交易失败 4:操作失败 5:提单失败
				} `json:"0"`
			} `json:"data"`
			//操作失败是 907 支付 独有的状态,泛指会员账号无法登入、
			//会员转账资料有误、会员余额不足等状态。
		} `json:"result"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.Code != 10000 {
		channelResp.Msg = payutils.ErrorMap[channelResp.Code]
		logx.WithContext(l.ctx).Errorf("支付查询渠道返回错误: %s: %s", channelResp.Code, channelResp.Msg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}
	Amount, errParse := strconv.ParseFloat(channelResp.Result.Data.Zero.Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}
	//
	orderStatus := "0"
	if channelResp.Result.Data.Zero.Status == 1 { //0:未处理 1:交易成功 	2:处理中 3:交易失败 4:操作失败 5:提单失败
		orderStatus = "1"
	} else if channelResp.Result.Data.Zero.Status == 3 || channelResp.Result.Data.Zero.Status == 4 || channelResp.Result.Data.Zero.Status == 5 {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: Amount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
