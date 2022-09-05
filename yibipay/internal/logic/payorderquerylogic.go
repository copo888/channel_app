package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/yibipay/internal/payutils"
	"github.com/copo888/channel_app/yibipay/internal/svc"
	"github.com/copo888/channel_app/yibipay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

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
	aesKey := "qHp8VxRtzQ7HpBfE"
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	timeStamp := strconv.FormatInt(time.Now().Unix(), 10)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	// 組請求參數
	// 組請求參數 FOR JSON
	dataInit := struct {
		MerchId       string `json:"merchantCode"`
		OrderId       string `json:"orderId"`
		TimeStamp     string `json:"timestamp"`
		TransactionId string `json:"transactionId"`
	}{
		MerchId:       channel.MerId,
		OrderId:       req.OrderNo,
		TimeStamp:     timeStamp,
		TransactionId: req.ChannelOrderNo,
	}
	dataBytes, err := json.Marshal(dataInit)
	params := utils.EnPwdCode(string(dataBytes), aesKey)
	sign := payutils.SortAndSignSHA256FromObj(dataInit, channel.MerKey)
	logx.WithContext(l.ctx).Infof("加签原串:%s，Encryption: %s，Signature: %s", string(dataBytes)+channel.MerKey, params, sign)
	data := struct {
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params"`    //参数密文
		Signature    string `json:"signature"` //参数签名(params + md5key)
	}{
		MerchantCode: channel.MerId,
		Params:       params,
		Signature:    sign,
	}
	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params, optional"`
		Sign         string `json:"signature"`
	}{}
	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}
	paramsDecode := utils.DePwdCode(channelResp.Params, aesKey)
	logx.WithContext(l.ctx).Infof("paramsDecode: %s", paramsDecode)
	channelResp2 := struct {
		//Code string `json:"code"`
		Data struct {
			Code          string `json:"code,optional"`
			Amount        string `json:"amount,optional"`
			CreatedAt     string `json:"createdAt,optional"`
			Success       bool   `json:"success,optional"`
			TransactionId string `json:"transactionId,optional"`
		} `json:"params,optional"`
		Message string `json:"message,optional"`
	}{}
	if err = json.Unmarshal([]byte(paramsDecode), &channelResp2); err != nil {
		logx.WithContext(l.ctx).Errorf("反序列化失败: ", err)
	}
	if channelResp2.Data.Code != "200" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp2.Message)
	}

	orderAmount, errParse := strconv.ParseFloat(channelResp2.Data.Amount, 64)
	if errParse != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	}

	orderStatus := "0"
	if channelResp2.Data.Success == true {
		orderStatus = "1"
	} else if channelResp2.Data.Success == false {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
