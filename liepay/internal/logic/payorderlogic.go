package logic

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/liepay/internal/payutils"
	"github.com/copo888/channel_app/liepay/internal/svc"
	"github.com/copo888/channel_app/liepay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
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

	// 檢查 userId
	if req.PayType == "YK" && len(req.UserId) == 0 {
		logx.WithContext(l.ctx).Errorf("userId不可为空 userId:%s", req.UserId)
		return nil, errorx.New(responsex.INVALID_USER_ID)
	}

	// 取值
	notifyUrl := l.svcCtx.Config.Server + "/api/pay-call-back"
	//notifyUrl = "http://b2d4-211-75-36-190.ngrok.io/api/pay-call-back"

	// 組請求參數 FOR JSON
	data := struct {
		MerchId      string `json:"mch_id"`
		MchOrderNo   string `json:"mch_order_no"`
		Amount       string `json:"amount"`
		Method       string `json:"method"`
		Format       string `json:"format"`
		NotifyUrl    string `json:"notify_url"`
		CallbackUrl  string `json:"callback_url"`
		RandomString string `json:"random_string"`
		BuyerName    string `json:"buyer_name"`
		Sign         string `json:"sign"`
	}{
		MerchId:      channel.MerId,
		MchOrderNo:   req.OrderNo,
		Amount:       req.TransactionAmount,
		Method:       req.ChannelPayType,
		Format:       "JSON",
		NotifyUrl:    notifyUrl,
		CallbackUrl:  notifyUrl,
		RandomString: fmt.Sprintf("%x", md5.Sum([]byte(req.OrderNo))),
	}

	// 加簽
	sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	data.Sign = sign
	data.BuyerName = req.UserId

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
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%v", channel.PayUrl, data)
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
		Status string `json:"status"`
		Msg    string `json:"msg, optional"`
		Data   struct {
			TransOrderNo   string `json:"trans_order_no"`
			Amount         string `json:"amount"`
			RealAmount     string `json:"real_amount"`
			Name           string `json:"name"`
			BankCardNumber string `json:"bank_card_number"`
			BankName       string `json:"bank_name"`
			BankBranch     string `json:"bank_branch"`
		} `json:"data"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, channelResp.Msg)
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
	if channelResp.Status != "success" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	// 若需回傳JSON 請自行更改
	amount, err2 := strconv.ParseFloat(channelResp.Data.RealAmount, 64)
	if err2 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
	}
	// 返回json
	receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
		CardName:   channelResp.Data.Name,
		CardNumber: channelResp.Data.BankCardNumber,
		BankName:   channelResp.Data.BankName,
		BankBranch: channelResp.Data.BankBranch,
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
		ChannelOrderNo: channelResp.Data.TransOrderNo,
		IsCheckOutMer:  true, // 自組收銀台回傳 true
	}, nil

	return
}
