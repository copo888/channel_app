package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/wtpay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/wtpay/internal/svc"
	"github.com/copo888/channel_app/wtpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderLogic {
	return ProxyPayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayOrderLogic) ProxyPayOrder(req *types.ProxyPayOrderRequest) (*types.ProxyPayOrderResponse, error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrder. channelName: %s, ProxyPayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	//channelBankMap, err2 := model2.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.ReceiptCardBankCode)
	//if err2 != nil || channelBankMap.MapCode == "" {
	//	logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.ReceiptCardBankCode, req.ReceiptCardBankName, channelBankMap.MapCode)
	//	return nil, errorx.New(responsex.BANK_CODE_INVALID, err2.Error(), "银行代码: "+req.ReceiptCardBankCode, "银行名称: "+req.ReceiptCardBankName)
	//}
	// 取值
	amount := utils.FloatMul(req.TransactionAmount, "100") // 單位:分
	// 組請求參數 FOR JSON
	data := struct {
		PlatformId  string `json:"platform_id"`
		ServiceId   string `json:"service_id"`
		PayoutClId  string `json:"payout_cl_id"`
		Amount      string `json:"amount"`
		NotifyUrl   string `json:"notify_url"`
		Name        string `json:"name"`
		Number      string `json:"number"`
		RequestTime string `json:"request_time"`
		Sign        string `json:"sign"`
	}{
		PlatformId:  channel.MerId,
		ServiceId:   "SVC0004",
		PayoutClId:  req.OrderNo,
		Amount:      fmt.Sprintf("%f", amount), // 單位:分
		NotifyUrl:   l.svcCtx.Config.Server + "/api/proxy-pay-call-back",
		Name:        req.ReceiptAccountName,
		Number:      req.ReceiptAccountNumber,
		RequestTime: strconv.FormatInt(time.Now().Unix(), 10),
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("代付下单请求地址:%s,代付請求參數:%#v", channel.ProxyPayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayUrl).Timeout(10).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			PayoutId string `json:"payout_id"`
		} `json:"data"`
	}{}

	if err := ChannelResp.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if channelResp.ErrorCode != "0000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg)
	}

	//組返回給backOffice 的代付返回物件
	resp := &types.ProxyPayOrderResponse{
		ChannelOrderNo: channelResp.Data.PayoutId,
		OrderStatus:    "",
	}

	return resp, nil
}
