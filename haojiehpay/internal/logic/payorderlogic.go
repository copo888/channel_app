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
	"github.com/copo888/channel_app/haojiehpay/internal/payutils"
	"github.com/copo888/channel_app/haojiehpay/internal/svc"
	"github.com/copo888/channel_app/haojiehpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/url"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %#v", l.svcCtx.Config.ProjectName, req)

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
	//notifyUrl = "https://dc98-211-75-36-190.jp.ngrok.io/api/pay-call-back"
	//timestamp := time.Now().Format("20060102150405")
	ip := utils.GetRandomIp()
	//randomID := utils.GetRandomString(32, utils.ALL, utils.MIX)
	appId := "145fd6be4e5b4187839447579d70a984"
	// 組請求參數
	data := url.Values{}
	//data.Set("mchId", channel.MerId)
	//data.Set("appId",)
	//data.Set("productId", req.ChannelPayType)
	//data.Set("mchOrderNo", req.OrderNo)
	//
	//data.Set("money", req.TransactionAmount)
	//data.Set("userId", randomID)
	//
	//data.Set("time", timestamp)
	//data.Set("notifyUrl", notifyUrl)
	//
	//data.Set("reType", "LINK")
	//data.Set("signType", "MD5")

	//mchId, _ := strconv.Atoi(channel.MerId)
	//productId, _ := strconv.Atoi(req.ChannelPayType)
	amount := utils.FloatMul(req.TransactionAmount, "100") // 單位:分
	amountInt := int(amount)
	// 組請求參數 FOR JSON
	dataJs := struct {
		MchId   string `json:"mchId"`
		AppId     string `json:"appId"`
		ProductId   string `json:"productId"`
		MchOrderNo string `json:"mchOrderNo"`
		Currency  string `json:"currency"`
		Amount    string    `json:"amount"`
		ClientIp  string  `json:"clientIp"`
		NotifyUrl string `json:"notifyUrl"`
		Subject   string `json:"subject"`
		Body      string `json:"body"`
		Param2    string `json:"param2"`
		Sign      string `json:"sign"`
	}{
		MchId:      channel.MerId,
		AppId:      appId,
		ProductId:  req.ChannelPayType,
		MchOrderNo: req.OrderNo,
		Currency:   strings.ToLower(req.Currency),
		Amount:     fmt.Sprintf("%d", amountInt), // 單位:分, // 單位:分
		ClientIp:   ip,
		NotifyUrl:  notifyUrl,
		Subject:    "COPO",
		Body:       "COPO",
		Param2:     req.UserId,
	}

	//if strings.EqualFold(req.JumpType, "json") {
	//	data.Set("reType", "INFO")
	//}

	// 加簽
	//sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	//data.Set("sign", sign)
	sign := payutils.SortAndSignFromObj(dataJs, channel.MerKey)
	dataJs.Sign = sign
	b, err := json.Marshal(dataJs)
	if err != nil {
		fmt.Println("error:", err)
	}
	data.Set("params", string(b))
	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%#v", channel.PayUrl, data)
	span := trace.SpanFromContext(l.ctx)
	// 若有證書問題 請使用
	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	//}
	//res, ChnErr := gozzle.Post(channel.PayUrl).Transport(tr).Timeout(20).Trace(span).Form(data)

	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).JSON(data)
	res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelResp := struct {
		RetCode    string `json:"retCode"`
		RetMsg     string `json:"retMsg, optional"`
		Sign    string `json:"sign, optional"`
		PayOrderId string `json:"payOrderId, optional"`
		PayParams struct {
			PayUrl string `json:"payUrl, optional"`
		} `json:"payParams, optional"`
	}{}

	// 返回body 轉 struct
	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	// 渠道狀態碼判斷
	if channelResp.RetCode != "SUCCESS" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.RetMsg)
	}

	// 若需回傳JSON 請自行更改
	//if strings.EqualFold(req.JumpType, "json") {
	//	amount, err2 := strconv.ParseFloat(channelResp.Money, 64)
	//	if err2 != nil {
	//		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err2.Error())
	//	}
	//	// 返回json
	//	receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
	//		CardName:   channelResp.PayInfo.Name,
	//		CardNumber: channelResp.PayInfo.Card,
	//		BankName:   channelResp.PayInfo.Bank,
	//		BankBranch: channelResp.PayInfo.Subbranch,
	//		Amount:     amount,
	//		Link:       "",
	//		Remark:     "",
	//	})
	//	if err3 != nil {
	//		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	//	}
	//	return &types.PayOrderResponse{
	//		PayPageType:    "json",
	//		PayPageInfo:    string(receiverInfoJson),
	//		ChannelOrderNo: "",
	//		IsCheckOutMer:  false, // 自組收銀台回傳 true
	//	}, nil
	//}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    channelResp.PayParams.PayUrl,
		ChannelOrderNo: "",
	}

	return
}
