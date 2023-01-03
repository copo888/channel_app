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
	"github.com/copo888/channel_app/wgopay88/internal/payutils"
	"github.com/copo888/channel_app/wgopay88/internal/svc"
	"github.com/copo888/channel_app/wgopay88/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
	"time"
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

	/** UserId 必填時使用1 **/
	if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"
	amount := utils.FloatMul(req.TransactionAmount, "100") // 單位:分
	amountInt := int(amount)
	// 組請求參數 FOR JSON
	data := struct {
		PlatformId  string `json:"platform_id"`
		ServiceId   string `json:"service_id"`
		PaymentClId string `json:"payment_cl_id"`
		Amount      string `json:"amount"`
		NotifyUrl   string `json:"notify_url"`
		Name        string `json:"name"`
		RequestTime string `json:"request_time"`
		Sign        string `json:"sign"`
	}{
		PlatformId:  channel.MerId,
		ServiceId:   req.ChannelPayType,
		PaymentClId: req.OrderNo,
		Amount:      fmt.Sprintf("%d", amountInt), // 單位:分
		NotifyUrl:   notifyUrl,
		Name:        req.UserId,
		RequestTime: strconv.FormatInt(time.Now().Unix(), 10),
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign

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
	jsonString, _ := json.Marshal(data)
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%s", channel.PayUrl, string(jsonString))
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			Link        string `json:"link"`
			PaymentInfo struct {
				Amount        float64 `json:"amount"`
				DisplayAmount float64 `json:"display_amount"`
				PaymentId     string  `json:"payment_id"`
				PaymentClId   string  `json:"payment_cl_id"`
				Receiver      struct {
					CardName   string `json:"card_name"`
					CardNumber string `json:"card_number"`
					BankName   string `json:"bank_name"`
					BankBranch string `json:"bank_branch"`
				} `json:"receiver"`
			} `json:"payment_info"`
		} `json:"data"`
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
		Content:   fmt.Sprintf("%+v", channelResp)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 渠道狀態碼判斷
	if channelResp.ErrorCode != "0000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.ErrorMsg)
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {

		amountDollar := utils.FloatDivF(channelResp.Data.PaymentInfo.Amount, 100) // 單位:分
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.Data.PaymentInfo.Receiver.CardName,
			CardNumber: channelResp.Data.PaymentInfo.Receiver.CardNumber,
			BankName:   channelResp.Data.PaymentInfo.Receiver.BankName,
			BankBranch: channelResp.Data.PaymentInfo.Receiver.BankBranch,
			Amount:     amountDollar,
			Link:       channelResp.Data.Link,
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}
		return &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: channelResp.Data.PaymentInfo.PaymentId,
			IsCheckOutMer:  false, // 自組收銀台回傳 true
		}, nil
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Data.Link,
		ChannelOrderNo: channelResp.Data.PaymentInfo.PaymentId,
	}

	return
}
