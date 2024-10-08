package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/apimodel/vo"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/tiansiapay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/tiansiapay/internal/svc"
	"github.com/copo888/channel_app/tiansiapay/internal/types"

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

	crypted, _ := base64.StdEncoding.DecodeString(req.Params)
	logx.WithContext(l.ctx).Infof("base64解密后的：%s", crypted)
	callBackBytes := payutils.AesDecrypt(crypted, []byte(l.svcCtx.Config.AesKey))
	logx.WithContext(l.ctx).Infof("解密后的明文：%s", string(callBackBytes))

	callBack := struct {
		OrderNo         string  `json:"orderNo, optional"`
		OrderStatus     int64   `json:"orderStatus, optional"`
		OrderAmount     float64 `json:"orderAmount, optional"`
		PaidAmount      float64 `json:"paidAmount, optional"`
		PlayerName      string  `json:"playerName, optional"`
		MerchantOrderId string  `json:"merchantOrderId, optional"`
		DepositName     string  `json:"depositName, optional"`
	}{}

	if err = json.Unmarshal(callBackBytes, &callBack); err != nil {
		return "fail", err
	}

	orderStatus := "1"
	if callBack.OrderStatus == 1 {
		orderStatus = "20"
	}

	payCallBackBO := bo.PayCallBackBO{
		PayOrderNo:     callBack.MerchantOrderId,
		ChannelOrderNo: callBack.OrderNo, // 渠道訂單號 (若无则填入->"CHN_" + orderNo)
		OrderStatus:    orderStatus, // 若渠道只有成功会回调 固定 20:成功; 訂單狀態(1:处理中 20:成功 )
		OrderAmount:    callBack.PaidAmount,
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


	result := struct {
		Code int64   `json:"code"`
		Msg string `json:"msg"`
	}{}

	result.Code = 200
	resultJson, _ := json.Marshal(result)
	return string(resultJson), nil
}
