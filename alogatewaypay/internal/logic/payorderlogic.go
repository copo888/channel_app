package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/alogatewaypay/internal/payutils"
	"github.com/copo888/channel_app/alogatewaypay/internal/service"
	"github.com/copo888/channel_app/alogatewaypay/internal/svc"
	"github.com/copo888/channel_app/alogatewaypay/internal/types"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/zeromicro/go-zero/core/logx"
	"io/ioutil"
	"net/http"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %+v", l.svcCtx.Config.ProjectName, req)
	//INDIA NETBANKING DEPOSIT1
	merchantAccount := "901720"
	PartnerControl := "4f8e98f980ca58374726d814d73546b6"

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
	//notifyUrl := "http://83d6-211-75-36-190.ngrok.io/api/pay-call-back"
	//amount, err := strconv.ParseFloat(req.TransactionAmount, 64)
	amounFloat := utils.FloatMul(req.TransactionAmount, "100")
	var ip string
	if len(req.SourceIp) > 0 {
		ip = req.SourceIp
	} else {
		ip = utils.GetRandomIp()
	}
	var control string
	// 組請求參數
	data := url.Values{}
	if strings.EqualFold(req.PayType, "UPI") {
		data.Set("merchant_account", channel.MerId)
		control = channel.MerKey
	} else if strings.EqualFold(req.PayType, "ND") {
		data.Set("merchant_account", merchantAccount)
		control = PartnerControl
	}
	data.Set("amount", fmt.Sprintf("%.f", amounFloat))
	data.Set("currency", "INR") //印度盧比
	data.Set("first_name", strings.ReplaceAll(req.UserId, " ", ""))
	data.Set("last_name", strings.ReplaceAll(req.UserId, " ", ""))
	data.Set("address1", strings.ReplaceAll(req.Address, " ", "")) //请商户传
	data.Set("city", req.City)                                     //请商户传
	data.Set("zip_code", req.ZipCode)                              //请商户传
	data.Set("country", req.Country)                               //请商户传(EX:IN)
	data.Set("phone", req.Phone)                                   //请商户传
	data.Set("email", req.Email)                                   //请商户传
	data.Set("merchant_order", req.OrderNo)
	data.Set("merchant_product_desc", "deposit")
	data.Set("return_url", req.PageUrl)
	keys := []string{"merchant_account", "amount", "currency", "first_name", "last_name", "address1", "city", "zip_code", "country", "phone", "email", "merchant_order", "merchant_product_desc", "return_url"}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(l.ctx, data, keys, control)
	data.Set("control", sign)
	data.Set("apiversion", "3")
	data.Set("version", "11")
	data.Set("ipaddress", ip)                //SourceIp
	data.Set("server_return_url", notifyUrl) //SourceIp

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
	logx.WithContext(l.ctx).Infof("支付下单请求地址:%s,支付請求參數:%+v", channel.PayUrl, data)
	//span := trace.SpanFromContext(l.ctx)

	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	requset, err := http.NewRequest("POST", channel.PayUrl, strings.NewReader(data.Encode()))
	requset.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, ChnErr := client.Do(requset)

	//pageUrl, _ := res.Location()
	pageUrl := res.Header.Values("Location")
	logx.WithContext(l.ctx).Infof("Status: %d", res.StatusCode)
	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
		msg := fmt.Sprintf("支付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
		service.DoCallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if res.StatusCode == 302 && pageUrl == nil {
		defer res.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		stringBody := string(bodyBytes)
		logx.WithContext(l.ctx).Errorf("Http status : %s,Http Body :%s", res.Status, stringBody)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, "status:302。未回传支付网址")
	} else if res.StatusCode == 200 {
		defer res.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		stringBody := string(bodyBytes)
		beginIndex := strings.Index(stringBody, "message")
		logx.WithContext(l.ctx).Errorf("Http status : %s,Http Body ErrorMsg :", res.Status, stringBody)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, stringBody[beginIndex+16:beginIndex+200])
	} else if res.StatusCode != 200 && res.StatusCode != 302 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status, res.Body)
		msg := fmt.Sprintf("支付提单，呼叫渠道返回Http状态码錯誤: '%s'，订单号： '%s'", res.Status, req.OrderNo)
		service.DoCallLineSendURL(l.ctx, l.svcCtx, msg)
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.StatusCode))
	}

	//res, ChnErr := gozzle.Post(channel.PayUrl).Timeout(20).Trace(span).Form(data)
	//if ChnErr != nil {
	//	logx.WithContext(l.ctx).Error("呼叫渠道返回錯誤: ", ChnErr.Error())
	//	msg := fmt.Sprintf("支付提单，呼叫渠道返回錯誤: '%s'，订单号： '%s'", ChnErr.Error(), req.OrderNo)
	//	service.DoCallLineSendURL(l.ctx, l.svcCtx, msg)
	//	return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	//} else if res.Status != "200" {
	//	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status, res.Body)
	//	msg := fmt.Sprintf("支付提单，呼叫渠道返回Http状态码錯誤: '%s'，订单号： '%s'", res.Status, req.OrderNo)
	//	service.DoCallLineSendURL(l.ctx, l.svcCtx, msg)
	//	return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %s", res.Status))
	//}

	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	//channelResp := struct {
	//	Code    string `json:"code"`
	//	Msg     string `json:"msg, optional"`
	//	Sign    string `json:"sign"`
	//	Money   string `json:"money"`
	//	OrderId string `json:"orderId"`
	//	PayUrl  string `json:"payUrl"`
	//	PayInfo struct {
	//		Name       string `json:"name"`
	//		Card       string `json:"card"`
	//		Bank       string `json:"bank"`
	//		Subbranch  string `json:"subbranch"`
	//		ExpiringAt string `json:"expiring_at"`
	//	} `json:"payInfo"`
	//}{}

	// 返回body 轉 struct
	//if err = res.DecodeJSON(&channelResp); err != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	//}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: req.MerchantId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo,
		LogType:   constants.RESPONSE_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", res.Body)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    pageUrl[0],
		ChannelOrderNo: "",
	}

	return
}
