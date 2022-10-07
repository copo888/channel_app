package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/yibipay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/yibipay/internal/svc"
	"github.com/copo888/channel_app/yibipay/internal/types"

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

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %+v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	timestamp := time.Now().Format("20060102150405")
	aesKey := "qHp8VxRtzQ7HpBfE"
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.WithContext(l.ctx).Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	dataInit := &Data{
		MerchantCode:    channel.MerId,
		MerchantId:      channel.MerId,
		Timestamp:       timestamp,
		WithdrawOrderId: req.OrderNo,
	}
	dataBytes, err := json.Marshal(dataInit)
	if err != nil {
		logx.Errorf("序列化失败: %s", err.Error())
	}
	params := utils.EnPwdCode(string(dataBytes), aesKey)
	// 加簽
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
		MerchantCode string `json:"merchantCode"`
		Params       string `json:"params, optional"`
		Sign         string `json:"signature"`
	}{}
	// 返回body 轉 struct
	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	}
	paramsDecode := utils.DePwdCode(channelQueryResp.Params, aesKey)
	logx.WithContext(l.ctx).Infof("paramsDecode: %s", paramsDecode)
	channelQueryResp2 := struct {
		Code string `json:"code"`
		Data struct {
			Amount        string `json:"amount,optional"`
			CreatedAt     string `json:"createdAt,optional"`
			CompletedAt   string `json:"completedAt,optional"`
			Fee           string `json:"fee,optional"`
			OrderStatus   string `json:"orderStatus,optional"` //提款订单的处理状态, 1确认成功,9确认失败,2处理中
			TransactionId string `json:"transactionId,optional"`
		} `json:"data,optional"`
		Message   string `json:"message,optional"`
		Timestamp string `json:"timestamp,optional"`
	}{}

	if err = json.Unmarshal([]byte(paramsDecode), &channelQueryResp2); err != nil {
		logx.WithContext(l.ctx).Errorf("反序列化失败: ", err)
	}
	if channelQueryResp2.Code != "200" {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s: %s", channelQueryResp2.Code, channelQueryResp2.Message)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp2.Message)
	}

	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1"
	if channelQueryResp2.Data.OrderStatus == "1" {
		orderStatus = "20"
	} else if channelQueryResp2.Data.OrderStatus == "9" {
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
