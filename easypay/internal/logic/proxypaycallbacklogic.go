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
	"github.com/copo888/channel_app/easypay/internal/svc"
	"github.com/copo888/channel_app/easypay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"time"
)

type ProxyPayCallBackLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewProxyPayCallBackLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayCallBackLogic {
	return ProxyPayCallBackLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *ProxyPayCallBackLogic) ProxyPayCallBack(req *types.ProxyPayCallBackRequest) (resp string, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayCallBack. channelName: %s, orderNo: %s, ProxyPayCallBackRequest: %+v", l.svcCtx.Config.ProjectName, req.OrderNo, req)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo:      channel.MerId,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OrderNo, //輸入COPO訂單號
		LogType:   constants.CALLBACK_FROM_CHANNEL,
		LogSource: constants.API_DF,
		Content:   fmt.Sprintf("%+v", req),
		TraceId:   l.traceID,
	}); err != nil {
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
	//if isSameSign := payutils.VerifySign(req.Sign, *req, channel.MerKey, l.ctx); !isSameSign {
	//	return "fail", errorx.New(responsex.INVALID_SIGN)
	//}

	//var orderAmount float64
	//if orderAmount, err = strconv.ParseFloat(req.Amount, 64); err != nil {
	//	return "fail", errorx.New(responsex.INVALID_SIGN)
	//}
	var status = "0" //渠道回調狀態(0:處理中1:成功2:失敗)
	if req.Status == "PAID" {
		status = "1"
	} else if strings.Index("CANCELLED", req.Status) > -1 {
		status = "2"
	}

	proxyPayCallBackBO := &bo.ProxyPayCallBackBO{
		ProxyPayOrderNo:     req.OrderNo,
		ChannelOrderNo:      "",
		ChannelResultAt:     time.Now().Format("20060102150405"),
		ChannelResultStatus: status,
		ChannelResultNote:   "",
		Amount:              req.Amount,
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
	logx.WithContext(l.ctx).Info("回调后资讯: ", res)
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
