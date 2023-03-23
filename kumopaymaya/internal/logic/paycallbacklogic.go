package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/apimodel/vo"
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
	"time"

	"github.com/copo888/channel_app/kumopaymaya/internal/svc"
	"github.com/copo888/channel_app/kumopaymaya/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayCallBackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayCallBackLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayCallBackLogic {
	return PayCallBackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayCallBackLogic) PayCallBack(req *types.PayCallBackRequest) (resp string, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayCallBack. channelName: %s, PayCallBackRequest: %+v", l.svcCtx.Config.ProjectName, req)

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo: "",
		//MerchantOrderNo: req.OrderNo,
		OrderNo:   req.OutTradeNo, //輸入COPO訂單號
		LogType:   constants.CALLBACK_FROM_CHANNEL,
		LogSource: constants.API_ZF,
		Content:   fmt.Sprintf("%+v", req)}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err != nil {
		return "fail", errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}

	// 檢查白名單
	if isWhite := utils.IPChecker(req.MyIp, channel.WhiteList); !isWhite {
		return "fail", errorx.New(responsex.IP_DENIED, "IP: "+req.MyIp)
	}

	// 把传送过去的 body 中的 (trade_no, amount, out_trade_no, state) + api_token + notify_token 做 md5

	//// 檢查驗簽
	//if isSameSign := payutils.VerifySign(req.Sign, *req, channel.MerKey + channel.MerId); !isSameSign {
	//	return "fail", errorx.New(responsex.INVALID_SIGN)
	//}
	// 檢查驗簽
	precise := payutils.GetDecimalPlaces(req.RequestAmount)
	requestAmount := strconv.FormatFloat(req.RequestAmount, 'f', precise, 64)

	source := fmt.Sprintf("amount=%s&out_trade_no=%s&request_amount=%s&state=%s&trade_no=%s%s%s",
		req.Amount, req.OutTradeNo, requestAmount, req.State, req.TradeNo, channel.MerKey, channel.MerId)
	sign := payutils.GetSign(source)
	logx.Info("verifySource: ", source)
	logx.Info("verifySign: ", sign)
	logx.Info("reqSign: ", req.Sign)
	if req.Sign != sign {
		return "fail", errorx.New(responsex.INVALID_SIGN)
	}


	var orderAmount float64
	if orderAmount, err = strconv.ParseFloat(req.Amount, 64); err != nil {
		return "fail", errorx.New(responsex.INVALID_AMOUNT)
	}

	//orderStatus := "1"
	//if req.TradeStatus == "1" {
	//	orderStatus = "20"
	//}

	payCallBackBO := bo.PayCallBackBO{
		PayOrderNo:     req.OutTradeNo,
		ChannelOrderNo: req.TradeNo, // 渠道訂單號 (若无则填入->"CHN_" + orderNo)
		OrderStatus:    "20",        // 若渠道只有成功会回调 固定 20:成功; 訂單狀態(1:处理中 20:成功 )
		OrderAmount:    orderAmount,
		CallbackTime:   time.Now().Format("20060102150405"),
	}

	/** 回調至 merchant service **/
	span := trace.SpanFromContext(l.ctx)
	// 組密鑰
	payKey, errk := utils.MicroServiceEncrypt(l.svcCtx.Config.ApiKey.PayKey, l.svcCtx.Config.ApiKey.PublicKey)
	if errk != nil {
		return "fail", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	url := fmt.Sprintf("%s:%d/dior/merchant-api/pay-call-back", l.svcCtx.Config.Merchant.Host, l.svcCtx.Config.Merchant.Port)
	res, errx := gozzle.Post(url).Timeout(20).Trace(span).Header("authenticationPaykey", payKey).JSON(payCallBackBO)
	logx.Info("回调后资讯: ", res)
	if errx != nil {
		return "err", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if res.Status() != 200 {
		return "err", errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("status:%d", res.Status()))
	}

	// 處理res
	payCallBackVO := vo.BoadminRespVO{}
	if err = res.DecodeJSON(&payCallBackVO); err != nil {
		return "err", err
	} else if payCallBackVO.Code != "0" {
		return "err", errorx.New(payCallBackVO.Code)
	}

	return "ok", nil
}