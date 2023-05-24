package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/koreapay/internal/payutils"
	"github.com/copo888/channel_app/koreapay/internal/service"
	"github.com/copo888/channel_app/koreapay/internal/svc"
	"github.com/copo888/channel_app/koreapay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}

	/** UserId 必填時使用 **/
	//if strings.EqualFold(req.PayType, "YK") && len(req.UserId) == 0 {
	//	logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
	//	return nil, errorx.New(responsex.INVALID_USER_ID)
	//}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl := "http://54d8-211-75-36-190.ngrok.io/api/pay-call-back"
	//timestamp := time.Now().Format("2006-01-02 15:04:05")
	//amountF, _ := strconv.ParseFloat(req.TransactionAmount, 64)

	// 組請求參數
	content := struct {
		MerchId         string `json:"merchantId"`
		OrderId         string `json:"transactionId"`
		TransactionType string `json:"transactionType"`
		PayType         string `json:"payMethod"`
		Currency        string `json:"currency"`
		Money           string `json:"amount"`
		NotifyUrl       string `json:"callback"`
		Response        string `json:"response"`
		PlayerId        string `json:"playerId,optional"`
		PlayerIp        string `json:"playerIp,optional"`
	}{
		MerchId:         channel.MerId,
		OrderId:         req.OrderNo,
		TransactionType: "D",
		PayType:         req.ChannelPayType,
		Currency:        "KRW",
		Money:           req.TransactionAmount,
		NotifyUrl:       notifyUrl,
		Response:        notifyUrl,
		PlayerId:        req.PlayerId,
		PlayerIp:        utils.GetRandomIp(),
	}

	// 組請求參數 FOR JSON
	data := struct {
		MerchId string `json:"merchantId"`
		Message string `json:"message"`
	}{
		MerchId: channel.MerId,
	}

	paramsJson, err := json.Marshal(content)
	paramsJsonStr := string(paramsJson)
	// 加簽
	aesData := payutils.AESEncrypt(strings.ReplaceAll(paramsJsonStr, " ", ""), []byte(channel.MerKey), l.svcCtx.Config.Channel.Pass1, l.svcCtx.Config.Channel.Pass2)
	encryptedString := base64.StdEncoding.EncodeToString(aesData)
	data.Message = encryptedString
	logx.Infof("paramsJsonStr: %s , data.Message(Encrypted): %s", paramsJsonStr, data.Message)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.DATA_REQUEST_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   data}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		msg := fmt.Sprintf("支付提单，呼叫渠道返回Http状态码錯誤: '%d'，订单号： '%s'", res.Status(), req.OrderNo)
		service.CallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		Status string `json:"status"`
		Errors string `json:"errors,optional"`
	}{}

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

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if strings.EqualFold(channelResp.Status, "fail") {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Errors)
	}

	channelResp1 := struct {
		Status               string `json:"status"`
		OrderId              string `json:"transactionId"`
		ReferenceId          string `json:"referenceId"`
		ReferenceNo          int64  `json:"referenceNo"`
		PayUrl               string `json:"redirectUrl"`
		DesignatedBankRemark string `json:"designatedBankRemark"`
		ExpiringAt           string `json:"dateExpired"`
		Currency             string `json:"currency"`
		Money                string `json:"deposit_amount"`
		Sign                 string `json:"signed"`
		PayInfo              struct {
			Name      string `json:"name,optional"`
			Card      string `json:"card,optional"`
			Bank      string `json:"bankName"`
			Subbranch string `json:"subbranch,optional"`
		} `json:"designatedBank"`
	}{}
	if err = res.DecodeJSON(&channelResp1); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 若需回傳JSON 請自行更改
	if strings.EqualFold(req.JumpType, "json") {
		amount, err2 := strconv.ParseFloat(channelResp1.Money, 64)
		if err2 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
		}
		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   channelResp1.PayInfo.Name,
			CardNumber: channelResp1.PayInfo.Card,
			BankName:   channelResp1.PayInfo.Bank,
			BankBranch: channelResp1.PayInfo.Subbranch,
			Amount:     amount,
			Link:       channelResp1.PayUrl,
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}
		return &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: channelResp1.ReferenceId,
			IsCheckOutMer:  false, // 自組收銀台回傳 true
		}, nil
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp1.PayUrl,
		ChannelOrderNo: channelResp1.ReferenceId,
	}

	return
}
