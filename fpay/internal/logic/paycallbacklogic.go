package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/apimodel/vo"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/fpay/internal/payutils"
	"github.com/copo888/channel_app/fpay/internal/svc"
	"github.com/copo888/channel_app/fpay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
	"time"
)

type PayCallBackLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayCallBackLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayCallBackLogic {
	return PayCallBackLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayCallBackLogic) PayCallBack(req *types.PayCallBackRequest) (resp string, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayCallBack. orderNo:%s, channelName: %s, PayCallBackRequest: %+v", req.MerchantOrderNum, l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err != nil {
		return "fail", errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: req.Merchant,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:     req.MerchantOrderNum, //輸入COPO訂單號
		ChannelCode: channel.Code,
		LogType:     constants.CALLBACK_FROM_CHANNEL,
		LogSource:   constants.API_ZF,
		Content:     fmt.Sprintf("%+v", req),
		TraceId:     l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	// 檢查白名單
	if isWhite := utils.IPChecker(req.MyIp, channel.WhiteList); !isWhite {
		return "fail", errorx.New(responsex.IP_DENIED, "IP: "+req.MyIp)
	}

	// 檢查驗簽
	//if isSameSign := payutils.VerifySign(req.Sign, *req, channel.MerKey, l.ctx); !isSameSign {
	//	return "fail", errorx.New(responsex.INVALID_SIGN)
	//}

	desOrder := struct {
		Amount              string `json:"amount"`
		Gateway             string `json:"gateway"`
		Status              string `json:"status"`
		MerchantOrderNum    string `json:"merchant_order_num"`
		MerchantOrderRemark string `json:"merchant_order_remark"`
	}{}

	desString, errDecode := payutils.AES256Decode(strings.ReplaceAll(req.Order, "\\n", ""), channel.MerKey, l.svcCtx.Config.HashIv)

	if errDecode != nil {
		return "fail", errDecode
	}
	logx.WithContext(l.ctx).Infof("desString:%s", desString)

	dd := strings.ReplaceAll(desString, "\x0f", "")
	dby := []byte(dd)
	errj := json.Unmarshal(dby, &desOrder)
	if errj != nil {
		logx.WithContext(l.ctx).Errorf("支付回调解析json失败, error : " + errj.Error())
		return "fail", errj
	}
	logx.WithContext(l.ctx).Infof("desOrder:%+v", desOrder)
	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		//MerchantNo: req.Merchant,
		//MerchantOrderNo: req.OrderNo,
		OrderNo:     req.MerchantOrderNum, //輸入COPO訂單號
		ChannelCode: channel.Code,
		LogType:     constants.CALLBACK_FROM_CHANNEL,
		LogSource:   constants.API_ZF,
		Content:     fmt.Sprintf("解密后:%+v", desOrder),
		TraceId:     l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	var orderAmount float64
	if orderAmount, err = strconv.ParseFloat(desOrder.Amount, 64); err != nil {
		return "fail", errorx.New(responsex.INVALID_AMOUNT)
	}

	orderStatus := "1"
	if desOrder.Status == "success" || desOrder.Status == "success_done" {
		orderStatus = "20"
	}

	payCallBackBO := bo.PayCallBackBO{
		PayOrderNo:     req.MerchantOrderNum,
		ChannelOrderNo: "CHN_" + desOrder.MerchantOrderNum, // 渠道訂單號 (若无则填入->"CHN_" + orderNo)
		OrderStatus:    orderStatus,                        // 若渠道只有成功会回调 固定 20:成功; 訂單狀態(1:处理中 20:成功 )
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
	logx.WithContext(l.ctx).Info("回调后资讯: ", res)
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

	return "success", nil
}
