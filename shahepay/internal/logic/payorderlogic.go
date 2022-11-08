package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/shahepay/internal/payutils"
	"github.com/copo888/channel_app/shahepay/internal/svc"
	"github.com/copo888/channel_app/shahepay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	timestamp := time.Now().Format(time.RFC3339)

	// 組請求參數 FOR JSON
	params := struct {
		AccountName     string `json:"account_name"`
		MerchantOrderId string `json:"merchant_order_id"`
		TotalAmount     string `json:"total_amount"`
		Timestamp       string `json:"timestamp"`
		NotifyUrl       string `json:"notify_url"`
		Subject         string `json:"subject"`
		GuestRealName   string `json:"guest_real_name"`
		PaymentMethod   string `json:"payment_method"`
		ForceMatching   bool   `json:"force_matching,optional"`
	}{
		AccountName:     channel.MerId,
		MerchantOrderId: req.OrderNo,
		TotalAmount:     req.TransactionAmount,
		Timestamp:       timestamp,
		NotifyUrl:       notifyUrl,
		Subject:         "订单",
		GuestRealName:   req.UserId,
		PaymentMethod:   req.ChannelPayType,
	}

	if strings.EqualFold(req.JumpType, "json") {
		params.ForceMatching = true
	}

	// 加簽
	paramsJson, _ := json.Marshal(params)
	signature := payutils.GetSign2(paramsJson, l.svcCtx.PrivateKey)
	// 組請求參數
	data := url.Values{}
	data.Set("data", string(paramsJson))
	data.Set("signature", signature)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", data)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%#v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)

	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() == 403 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, fmt.Sprintf("Error HTTP Status: %d, %s", res.Status(), string(res.Body())))
	} else if res.Status() < 200 && res.Status() >= 300 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		ID          string `json:"id"`
		PaymentUrl  string `json:"payment_url, optional"`
		TotalAmount string `json:"total_amount"`
		BankAccount struct {
			Name           string `json:"name"`
			AccountName    string `json:"account_name"`
			BankName       string `json:"bank_name"`
			BankBranchName string `json:"bank_branch_name"`
		} `json:"bank_account,optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", channelResp),
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}
	if strings.EqualFold(req.JumpType, "json") {
		var amount float64
		if amount, err = strconv.ParseFloat(channelResp.TotalAmount, 64); err != nil {
			logx.WithContext(l.ctx).Errorf("反卡讯息金额错误:%s", err.Error())
		}
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.BankAccount.Name,
			CardNumber: channelResp.BankAccount.AccountName,
			BankName:   channelResp.BankAccount.BankName,
			BankBranch: channelResp.BankAccount.BankBranchName,
			Amount:     amount,
			Link:       "",
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}

		resp = &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: channelResp.ID,
		}
	} else {
		resp = &types.PayOrderResponse{
			PayPageType:    "url",
			PayPageInfo:    channelResp.PaymentUrl,
			ChannelOrderNo: channelResp.ID,
		}
	}

	return
}
