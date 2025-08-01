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
	"github.com/copo888/channel_app/ui2pay/internal/payutils"
	"github.com/copo888/channel_app/ui2pay/internal/service"
	"github.com/copo888/channel_app/ui2pay/internal/svc"
	"github.com/copo888/channel_app/ui2pay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
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
	//if strings.EqualFold(req.PayType, "PP") && len(req.UserId) == 0 {
	//	logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
	//	return nil, errorx.New(responsex.INVALID_USER_ID)
	//} else if (strings.EqualFold(req.PayType, "PP") || strings.EqualFold(req.PayType, "YK")) && len(req.BankCode) == 0 {
	//	logx.WithContext(l.ctx).Errorf("BankCode不可为空 BankCode:%s", req.UserId)
	//	return nil, errorx.New(responsex.BANK_CODE_EMPTY)
	//} else if (strings.EqualFold(req.PayType, "PP") || strings.EqualFold(req.PayType, "YK")) && len(req.BankAccount) == 0 {
	//	logx.WithContext(l.ctx).Errorf("BankAccount不可为空 BankAccount:%s", req.BankAccount)
	//	return nil, errorx.New(responsex.BANK_ACCOUNT_EMPTY)
	//}

	//channelBankMap, err2 := model.NewChannelBank(l.svcCtx.MyDB).GetChannelBankCode(l.svcCtx.MyDB, channel.Code, req.BankCode)
	//if err2 != nil { //BankName空: COPO沒有對應銀行(要加bk_banks)，MapCode為空: 渠道沒有對應銀行代碼(要加ch_channel_banks)
	//	logx.WithContext(l.ctx).Errorf("銀行代碼抓取資料錯誤:%s", err2.Error())
	//	return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode, "银行名称: "+req.BankAccount, "渠道Map名称: "+channelBankMap.MapCode)
	//} else if channelBankMap.BankName == "" || channelBankMap.MapCode == "" {
	//	logx.WithContext(l.ctx).Errorf("银行代码: %s,银行名称: %s,渠道银行代码: %s", req.BankCode, req.BankAccount, channelBankMap.MapCode)
	//	return nil, errorx.New(responsex.BANK_CODE_INVALID, "银行代码: "+req.BankCode, "银行名称: "+req.BankAccount, "渠道Map名称: "+channelBankMap.MapCode)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"

	// 組請求參數 FOR JSON
	data := struct {
		MerchantNo       string `json:"merchantNo"`
		MerchantUplineNo string `json:"merchantUplineNo"` //商户上线号码
		OrderId          string `json:"orderNo"`
		Amt              string `json:"amt"`
		NotifyUrl        string `json:"noticeUrl"`
		Username         string `json:"username"`
		Sign             string `json:"sign"`
		//BankCode         string `json:"bankCode"`
		//AccName          string `json:"accName"`
		//AccNumber        string `json:"accNumber"`
	}{
		MerchantNo:       channel.MerId,
		MerchantUplineNo: "uipay",
		OrderId:          req.OrderNo,
		Amt:              req.TransactionAmount,
		NotifyUrl:        notifyUrl,
		Username:         "Copo",
		//BankCode:         channelBankMap.MapCode, //channelBankMap.MapCode,
		//AccName:          req.UserId,
		//AccNumber:        req.BankAccount,
	}

	sign := payutils.SortAndSignFromObj(data, channel.MerKey, l.ctx)
	data.Sign = sign

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		ChannelCode:     channel.Code,
		OrderNo:         req.OrderNo,
		LogType:         constants.DATA_REQUEST_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         data,
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	var url string
	if strings.EqualFold(req.PayType, "PP") {
		url = channel.PayUrl
	} else if req.PayType == "YK" {
		url = channel.PayQueryUrl
	}
	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", url, data)
	span := trace.SpanFromContext(l.ctx)
	// 若有證書問題 請使用
	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(20).Trace(span).Form(data)
	var ChnErr error
	var res *gozzle.Response
	res, ChnErr = gozzle.Post(url).Timeout(20).Trace(span).JSON(data)

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

		service.DoCallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫'%s'渠道返回Http状态码錯誤: '%d'，订单号： '%s'", channel.Name, res.Status(), req.OrderNo)
		service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: msg})

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

		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Code         int    `json:"code"`
		Msg          string `json:"msg"`
		ResponseTime int    `json:"responseTime,optional"`
		Detail       struct {
			PayURL   string  `json:"PayURL,optional"`
			Amt      float64 `json:"Amt,optional"`
			ApplyAmt string  `json:"ApplyAmt,optional"`
		} `json:"detail,optional"`
		CheckstandInfo struct {
			BankAccount     string `json:"bankAccount,optional"`
			BankCode        string `json:"bankCode,optional"`
			BankName        string `json:"bankName,optional"`
			BankAccountName string `json:"bankAccountName,optional"`
		} `json:"checkstandInfo,optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 渠道狀態碼判斷
	if channelResp.Code != 0 {
		// 寫入交易日志
		if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
			MerchantNo:       req.MerchantId,
			ChannelCode:      channel.Code,
			MerchantOrderNo:  req.MerchantOrderNo,
			OrderNo:          req.OrderNo,
			LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
			LogSource:        constants.API_ZF,
			Content:          fmt.Sprintf("%+v", channelResp),
			TraceId:          l.traceID,
			ChannelErrorCode: fmt.Sprintf("%d", channelResp.Code),
		}); err != nil {
			logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
		}
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
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

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		isCheckOutMer := false // COPO自組收銀台回傳 true
		amount, _ := strconv.ParseFloat(channelResp.Detail.ApplyAmt, 64)
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp.CheckstandInfo.BankAccountName,
			CardNumber: channelResp.CheckstandInfo.BankAccount,
			BankName:   channelResp.CheckstandInfo.BankName,
			BankBranch: channelResp.CheckstandInfo.BankCode,
			Amount:     amount,
			Link:       channelResp.Detail.PayURL,
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}
		return &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: "",
			IsCheckOutMer:  isCheckOutMer, // 自組收銀台回傳 true
		}, nil
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.Detail.PayURL,
		ChannelOrderNo: "",
	}

	return
}
