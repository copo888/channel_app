package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/mashangpay/internal/payutils"
	"github.com/copo888/channel_app/mashangpay/internal/svc"
	"github.com/copo888/channel_app/mashangpay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	// 組請求參數

	// 組請求參數 FOR JSON
	data := struct {
		MerchantCode    string `json:"merchantCode"`
		MerchantOrderId string `json:"merchantOrderId"`
		Sign            string `json:"sign"`
	}{
		MerchantCode:    channel.MerId,
		MerchantOrderId: req.OrderNo,
		//sign: "MD5",
	}

	// 加簽 JSON
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%+v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	Resp := struct {
		Result bool `json:"result"`
	}{}

	if err = json.Unmarshal(res.Body(), &Resp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}
	if Resp.Result != true {
		// 渠道回覆處理 [請依照渠道返回格式 自定義]
		channelResp := struct {
			Result   bool `json:"result"`
			ErrorMsg struct {
				Code     int    `json:"code"`
				ErrorMsg string `json:"errorMsg"`
				Descript string `json:"descript"`
			} `json:"errorMsg"`
		}{}
		if err = json.Unmarshal(res.Body(), &channelResp); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}

		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg.Descript)
	} else {
		channelResp := struct {
			Result   bool   `json:"result"`
			ErrorMsg string `json:"errorMsg, optional"`
			Data     struct {
				GamerOrderId    string `json:"gamerOrderId"`
				MerchantOrderId string `json:"merchantOrderId"`
				CurrencyCode    string `json:"currencyCode"`
				PaymentTypeCode string `json:"paymentTypeCode"`
				Amount          string `json:"amount"`
				Status          string `json:"status"`
			} `json:"data"`
		}{}
		if err = json.Unmarshal(res.Body(), &channelResp); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}
		amountFlo, _ := strconv.ParseFloat(channelResp.Data.Amount, 64)
		orderStatus := "0"
		if channelResp.Data.Status == "Success" {
			orderStatus = "1"
		} else if channelResp.Data.Status == "Failed" || channelResp.Data.Status == "Unpaid" {
			orderStatus = "2"
		}

		resp = &types.PayOrderQueryResponse{
			OrderAmount: amountFlo,
			OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
		}

	}

	// 渠道回覆處理
	//orderAmount, errParse := strconv.ParseFloat(channelResp.Money, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	return
}
