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
	"github.com/copo888/channel_app/jstpay4/internal/payutils"
	"github.com/copo888/channel_app/jstpay4/internal/service"
	"github.com/copo888/channel_app/jstpay4/internal/svc"
	"github.com/copo888/channel_app/jstpay4/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s,orderNo: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req.OrderNo, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	/** UserId 必填時使用 **/
	if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "https://b345-211-75-36-190.ngrok-free.app/api/pay-call-back"
	timestamp := time.Now().Format("20060102150405")
	amountFloat, _ := strconv.ParseFloat(req.TransactionAmount, 64)
	//ip := utils.GetRandomIp()
	//randomID := utils.GetRandomString(12, utils.ALL, utils.MIX)

	if amountFloat != math.Trunc(amountFloat) {
		logx.WithContext(l.ctx).Errorf("金额必须为整数, 请求金额:%s", req.TransactionAmount)
		return nil, errorx.New(responsex.INVALID_AMOUNT)
	}
	transactionAmount := strconv.FormatFloat(amountFloat, 'f', 0, 64)
	var bankCode string
	if req.ChannelPayType == "PA004" {
		bankCode = "602"
	} else if req.ChannelPayType == "PA003" || req.ChannelPayType == "PA002" {
		bankCode = "600"
	} else if req.ChannelPayType == "PA001" {
		bankCode = "001"
	}
	// 組請求參數
	data := url.Values{}
	data.Set("MERCHANT_ID", channel.MerId)
	data.Set("TRANSACTION_AMOUNT", transactionAmount)
	data.Set("TRANSACTION_ISSUE_TYPE", "API")
	data.Set("BANK_ACCOUNT_NAME", req.UserId)
	data.Set("BANK_ACCOUNT_NO", "12345678")
	data.Set("BANK_CODE", bankCode)
	data.Set("TRANSACTION_NUMBER", req.OrderNo)
	data.Set("SUBMIT_TIME", timestamp)
	data.Set("NOTIFY_URL", notifyUrl)
	data.Set("PAY_TYPE", req.ChannelPayType)
	data.Set("VERSION", "1")

	if req.JumpType == "json" {
		data.Set("RETURN_TYPE", "1")
	} else {
		data.Set("RETURN_TYPE", "2")
	}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey, l.ctx)
	data.Set("SIGNED_MSG", sign)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		OrderNo:         req.OrderNo,
		ChannelCode:     channel.Code,
		LogType:         constants.DATA_REQUEST_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         data}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	// 若有證書問題 請使用
	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(20).Trace(span).Form(data)

	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回錯誤: '%s'，订单号： '%s'", channel.Name, ChnErr.Error(), req.OrderNo)

		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          ChnErr.Error(),
			TraceId:          l.traceID,
			ChannelErrorCode: ChnErr.Error(),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}

		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          string(res.Body()),
			TraceId:          l.traceID,
			ChannelErrorCode: strconv.Itoa(res.Status()),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code string `json:"code"`
		Msg  string `json:"message, optional"`
		Data string `json:"data, optional"`
	}{}

	channelResp2 := struct {
		Code string `json:"code"`
		Msg  string `json:"message, optional"`
		Data struct {
			BankCode      string `json:"bankCode"`
			BranchName    string `json:"branchName"`
			BankName      string `json:"bankName"`
			AccountNumber string `json:"accountNumber"`
			AccountName   string `json:"accountName"`
		}
	}{}

	if req.JumpType == "json" {
		// 返回body 轉 struct
		if err = res.DecodeJSON(&channelResp2); err != nil {
			return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
		}
		//寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:      req.MerchantId,
			ChannelCode:     channel.Code,
			MerchantOrderNo: req.MerchantOrderNo,
			OrderNo:         req.OrderNo,
			LogType:         constants.RESPONSE_FROM_CHANNEL,
			LogSource:       constants.API_ZF,
			Content:         fmt.Sprintf("%+v", channelResp),
			TraceId:         l.traceID,
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}
		// 渠道狀態碼判斷
		if channelResp2.Code != "G_00001" {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
		}
		amount, err2 := strconv.ParseFloat(req.TransactionAmount, 64)
		if err2 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
		}
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp2.Data.AccountName,
			CardNumber: channelResp2.Data.AccountNumber,
			BankName:   channelResp2.Data.BankName,
			BankBranch: channelResp2.Data.BranchName,
			Amount:     amount,
			Link:       "",
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}
		return &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: "",
			IsCheckOutMer:  false, // 自組收銀台回傳 true
		}, nil
	} else {
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
			TraceId:   l.traceID,
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}
		// 渠道狀態碼判斷
		if channelResp.Code != "G_00001" {

			// 寫入交易日志
			if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
				MerchantNo:       req.MerchantId,
				ChannelCode:      channel.Code,
				MerchantOrderNo:  req.MerchantOrderNo,
				OrderNo:          req.OrderNo,
				LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
				LogSource:        constants.API_ZF,
				Content:          fmt.Sprintf("%+v", channelResp.Msg),
				TraceId:          l.traceID,
				ChannelErrorCode: fmt.Sprintf("%s", channelResp.Code),
			}); err != nil {
				logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
			}

			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
		}

		resp = &types.PayOrderResponse{
			PayPageType:    "url",
			PayPageInfo:    channelResp.Data,
			ChannelOrderNo: "",
		}
		return
	}
}
