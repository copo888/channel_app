package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/kumopaymaya/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
	"time"

	"github.com/copo888/channel_app/kumopaymaya/internal/svc"
	"github.com/copo888/channel_app/kumopaymaya/internal/types"

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

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo:      channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OutTradeNo, //輸入COPO訂單號
		LogType:   constants.CALLBACK_FROM_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", req)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

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
	source := ""
	if len(req.Errors) > 0 {
		source = fmt.Sprintf("amount=%s&errors=%s&out_trade_no=%s&state=%s&trade_no=%s%s%s",
			req.Amount, req.Errors, req.OutTradeNo, req.State, req.TradeNo, channel.MerKey, channel.MerId)
	} else {
		source = fmt.Sprintf("amount=%s&out_trade_no=%s&state=%s&trade_no=%s%s%s",
			req.Amount, req.OutTradeNo, req.State, req.TradeNo, channel.MerKey, channel.MerId)
	}

	sign := payutils.GetSign(source)
	logx.Info("verifySource: ", source)
	logx.Info("verifySign: ", sign)
	logx.Info("reqSign: ", req.Sign)
	if req.Sign != sign {
		return "fail", errorx.New(responsex.INVALID_SIGN)
	}

	var orderAmount float64
	if orderAmount, err = strconv.ParseFloat(req.Amount, 64); err != nil {
		return "fail", errorx.New(responsex.INVALID_SIGN)
	}
	var status = "0" //渠道回調狀態(0:處理中1:成功2:失敗)
	if req.State == "completed" {
		status = "1"
	} else if strings.Index("reject,failed,refund", req.State) > -1 {
		status = "2"
	}

	proxyPayCallBackBO := &bo.ProxyPayCallBackBO{
		ProxyPayOrderNo:     req.OutTradeNo,
		ChannelOrderNo:      req.TradeNo,
		ChannelResultAt:     time.Now().Format("20060102150405"),
		ChannelResultStatus: status,
		ChannelResultNote:   req.Errors,
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

	return "ok", nil
}
