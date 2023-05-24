package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/koreapay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"

	"github.com/copo888/channel_app/koreapay/internal/svc"
	"github.com/copo888/channel_app/koreapay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayCallBackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayCallBackLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayCallBackLogic {
	return ProxyPayCallBackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayCallBackLogic) ProxyPayCallBack(req *types.ProxyPayCallBackRequest) (resp string, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayCallBack. channelName: %s, ProxyPayCallBackRequest: %+v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err != nil {
		return "fail", errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}

	//檢查白名單
	if isWhite := utils.IPChecker(req.Ip, channel.WhiteList); !isWhite {
		logx.WithContext(l.ctx).Errorf("IP: " + req.Ip)
		return "fail", errorx.New(responsex.IP_DENIED, "IP: "+req.Ip)
	}

	encryptedMsg, _ := base64.StdEncoding.DecodeString(req.Message)
	decryptedText := payutils.AESDecrypt(encryptedMsg, []byte(channel.MerKey), l.svcCtx.Config.Channel.Pass1, l.svcCtx.Config.Channel.Pass2)
	var MessageData ProxyMessage
	if err1 := json.Unmarshal(decryptedText, &MessageData); err1 != nil {
		return "", errorx.New(responsex.PARAMETER_TYPE_ERROE, err.Error())
	}

	logx.Infof("解密后资料: %+v", MessageData)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo:      channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderId, //輸入COPO訂單號
		LogType:   constants.CALLBACK_FROM_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", req)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	var orderAmount float64
	if orderAmount, err = strconv.ParseFloat(MessageData.Amount, 64); err != nil {
		return "fail", errorx.New(responsex.INVALID_SIGN)
	}
	var status = "0"             //渠道回調狀態(0:處理中1:成功2:失敗)
	if MessageData.Status == 1 { //1: Completed	2: Rejected	4: Refunded
		status = "1"
	} else if MessageData.Status == 2 || MessageData.Status == 4 {
		status = "2"
	}

	proxyPayCallBackBO := &bo.ProxyPayCallBackBO{
		ProxyPayOrderNo:     req.OrderId,
		ChannelOrderNo:      "",
		ChannelResultAt:     time.Now().Format("20060102150405"),
		ChannelResultStatus: status,
		ChannelResultNote:   MessageData.StatusDesc,
		Amount:              orderAmount,
		ChannelCharge:       0,
		UpdatedBy:           "",
	}

	// call boadmin callback api
	span := trace.SpanFromContext(l.ctx)
	payKey, errk := utils.MicroServiceEncrypt(l.svcCtx.Config.ApiKey.ProxyKey, l.svcCtx.Config.ApiKey.PublicKey)
	if errk != nil {
		return "fail", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//BoProxyRespVO := &vo.BoadminProxyRespVO{}
	url := fmt.Sprintf("%s:%d/dior/merchant-api/proxy-call-back", l.svcCtx.Config.Merchant.Host, l.svcCtx.Config.Merchant.Port)

	res, errx := gozzle.Post(url).Timeout(20).Trace(span).Header("authenticationProxykey", payKey).JSON(proxyPayCallBackBO)
	logx.Info("回调后资讯: ", res)
	if errx != nil {
		logx.WithContext(l.ctx).Error(errx.Error())
		return "fail", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if res.Status() != 200 {
		return "fail", errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("status:%d", res.Status()))
	}
	//else if errDecode:= res.DecodeJSON(BoProxyRespVO); errDecode!=nil {
	//  return "fail",errorx.New(responsex.DECODE_JSON_ERROR)
	//} else if BoProxyRespVO.Code != "000"{
	//	return "fail",errorx.New(BoProxyRespVO.Message)
	//}

	replyStruct := struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{
		Status:  "success",
		Message: "",
	}
	replyByte, _ := json.Marshal(replyStruct)
	return string(replyByte), nil
}

type ProxyMessage struct {
	ReferenceId         string `json:"referenceId"`
	Status              int    `json:"status"`
	StatusDesc          string `json:"statusDesc"`
	RequestDate         string `json:"requestDate"`
	TransactionType     string `json:"transactionType"`
	TransactionTypeDesc string `json:"transactionTypeDesc"`
	Currency            string `json:"currency"`
	Amount              string `json:"amount"`
	TransactionDate     string `json:"transactionDate"`
	Remark              string `json:"remark"`
	FrombankName        string `json:"frombankName,optional"`      // 渠道成功状态，返回参数
	BankTransactionId   string `json:"bankTransactionId,optional"` // 渠道成功状态，返回参数
	TransactionAmount   string `json:"transactionAmount,optional"` // 渠道成功状态，返回参数
	TransactionFee      string `json:"transactionFee,optional"`    // 渠道成功状态，返回参数
	ServiceFee          string `json:"serviceFee,optional"`        // 渠道成功状态，返回参数
	Adjustment          struct {
		AdjustedDate             string `json:"adjustedDate"`
		AdjustedReason           string `json:"adjustedReason"`
		ServiceFeeBeforeAdjusted string `json:"serviceFeeBeforeAdjusted"`
	} `json:"adjustment,optional"` // 渠道成功状态，返回参数
	RefundedReason string `json:"refundedReason,optional"` // 渠道失败状态，返回参数
	AdjustedDate   string `json:"adjustedDate,optional"`   // 渠道失败状态，返回参数
	adjustedReason string `json:"adjustedReason,optional"` // 渠道失败状态，返回参数
}
