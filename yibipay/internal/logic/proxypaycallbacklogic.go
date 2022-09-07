package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/yibipay/internal/svc"
	"github.com/copo888/channel_app/yibipay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"time"
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
	aesKey := "qHp8VxRtzQ7HpBfE"
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
	// 檢查驗簽
	//if isSameSign := payutils.VerifySign(req.Sign, *req, channel.MerKey); !isSameSign {
	//	return "fail", errorx.New(responsex.INVALID_SIGN)
	//}

	//解密
	paramsDecode := utils.DePwdCode(req.Params, aesKey)
	logx.WithContext(l.ctx).Infof("解密后资料，paramsDecode: %s", paramsDecode)

	//反序列化
	chnResp := &channelProxyResp{}
	if err = json.Unmarshal([]byte(paramsDecode), chnResp); err != nil {
		logx.WithContext(l.ctx).Errorf("反序列化失败: ", err)
	}

	var status = "0"                     //渠道回調狀態(0:處理中1:成功2:失敗)
	if chnResp.Data.OrderStatus == "1" { //{ 0初始化 1确认成功,9确认失败,2处理中
		status = "1"
	} else if chnResp.Data.OrderStatus == "9" {
		status = "2"
	}

	var orderAmount float64
	if orderAmount, err = strconv.ParseFloat(chnResp.Data.Amount, 64); err != nil {
		return "fail", errorx.New(responsex.INVALID_AMOUNT)
	}

	proxyPayCallBackBO := &bo.ProxyPayCallBackBO{
		ProxyPayOrderNo:     chnResp.Data.WithdrawOrderId,
		ChannelOrderNo:      chnResp.Data.TransactionId,
		ChannelResultAt:     time.Now().Format("20060102150405"),
		ChannelResultStatus: status,
		//ChannelResultNote:   req.StatusStr,
		Amount:        orderAmount,
		ChannelCharge: 0,
		UpdatedBy:     "",
	}

	// call boadmin callback api
	span := trace.SpanFromContext(l.ctx)
	payKey, errk := utils.MicroServiceEncrypt(l.svcCtx.Config.ApiKey.PayKey, l.svcCtx.Config.ApiKey.PublicKey)
	if errk != nil {
		return "fail", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//BoProxyRespVO := &vo.BoadminProxyRespVO{}
	url := fmt.Sprintf("%s:%d/dior/merchant-api/proxy-call-back", l.svcCtx.Config.Merchant.Host, l.svcCtx.Config.Merchant.Port)

	res, errx := gozzle.Post(url).Timeout(20).Trace(span).Header("authenticationPaykey", payKey).JSON(proxyPayCallBackBO)
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

	return "success", nil
}

type channelProxyResp struct {
	Code string `json:"code"`
	Data struct {
		Amount          string `json:"amount,optional"`
		CompletedAt     string `json:"completedAt,optional"`
		CreatedAt       string `json:"createdAt,optional"`
		Currency        string `json:"currency,optional"`
		Fee             string `json:"fee,optional"`
		OrderStatus     string `json:"orderStatus,optional"` //0初始化 1确认成功,9确认失败,2处理中
		TransactionId   string `json:"transactionId,optional"`
		WalletAddress   string `json:"walletAddress,optional"`
		WithdrawOrderId string `json:"withdrawOrderId,optional"`
		TransactionHash string `json:"transactionHash,optional"`
	} `json:"data,optional"`
	MerchantCode string `json:"merchantCode"`
	Message      string `json:"message,optional"`
	Request      struct {
		MerchantCode string `json:"merchantCode,optional"`
		MerchantId   string `json:"merchantId,optional"`
		Timestamp    string `json:"timestamp,optional"`
	} `json:"request,optional"`
	Timestamp string `json:"timestamp,optional"`
}
