package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/apimodel/bo"
	"github.com/copo888/channel_app/common/apimodel/vo"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/yunshengpay/internal/svc"
	"github.com/copo888/channel_app/yunshengpay/internal/types"

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

	logx.WithContext(l.ctx).Infof("Enter PayCallBack. channelName: %s, PayCallBackRequest: %#v", l.svcCtx.Config.ProjectName, req)

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

	// 檢查驗簽
	//if isSameSign := payutils.VerifySign(req.Sign, *req, channel.MerKey); !isSameSign {
	//	return "fail", errorx.New(responsex.INVALID_SIGN)
	//}

	channelResp := struct {
		Code    int     `json:"code, optional"`
		Msg     string  `json:"msg, optional"`
		MerchId string  `json:"merchantId, optional"`
		Money   float64 `json:"amount, optional"`
		Fee     float64 `json:"fee, optional"`
		TradeNo string  `json:"transId, optional"`
		OrderNo string  `json:"orderId, optional"`
		Status  int     `json:"status, optional"`
	}{}

	response := utils.DePwdCode(req.Data, channel.MerKey)
	logx.WithContext(l.ctx).Infof("返回解密: %s", response)

	if err = json.Unmarshal([]byte(response), &channelResp); err != nil {
		return "", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	orderStatus := "1"
	if channelResp.Status == 0 { //0:为订单完成成功；11:取消
		orderStatus = "20"
	}

	payCallBackBO := bo.PayCallBackBO{
		PayOrderNo:     channelResp.OrderNo,
		ChannelOrderNo: channelResp.TradeNo, // 渠道訂單號 (若无则填入->"CHN_" + orderNo)
		OrderStatus:    orderStatus,         // 若渠道只有成功会回调 固定 20:成功; 訂單狀態(1:处理中 20:成功)
		OrderAmount:    channelResp.Money,
		CallbackTime:   time.Now().Format("20060102150405"),
	}

	/** 回調至 merchant service **/
	span := trace.SpanFromContext(l.ctx)
	// 組密鑰
	payKey, errk := utils.MicroServiceEncrypt(l.svcCtx.Config.ApiKey.PayKey, l.svcCtx.Config.ApiKey.PublicKey)
	if errk != nil {
		resultJson, _ := json.Marshal(Result{
			Code: -1,
			Msg:  errk.Error(),
		})
		return string(resultJson), errk
	}

	url := fmt.Sprintf("%s:%d/dior/merchant-api/pay-call-back", l.svcCtx.Config.Merchant.Host, l.svcCtx.Config.Merchant.Port)
	res, errx := gozzle.Post(url).Timeout(20).Trace(span).Header("authenticationPaykey", payKey).JSON(payCallBackBO)
	logx.Info("回调后资讯: ", res)
	if errx != nil {
		resultJson, _ := json.Marshal(Result{
			Code: -1,
			Msg:  errx.Error(),
		})
		return string(resultJson), errx
	} else if res.Status() != 200 {
		resultJson, err := json.Marshal(Result{
			Code: -1,
			Msg:  fmt.Sprintf("status error:%d", res.Status()),
		})
		return string(resultJson), err
	}

	// 處理res
	payCallBackVO := vo.BoadminRespVO{}
	if err = res.DecodeJSON(&payCallBackVO); err != nil {
		resultJson, err := json.Marshal(Result{
			Code: -1,
			Msg:  err.Error(),
		})
		return string(resultJson), err
	} else if payCallBackVO.Code != "0" {
		resultJson, _ := json.Marshal(Result{
			Code: -1,
			Msg:  err.Error(),
		})
		return string(resultJson), errorx.New(payCallBackVO.Code)
	}

	resultJson, err := json.Marshal(Result{
		Code: 0,
		Msg:  "成功",
	})
	return string(resultJson), nil
}

type ChannelResp struct {
	Code    int     `json:"code, optional"`
	Msg     string  `json:"msg, optional"`
	MerchId string  `json:"merchantId, optional"`
	Money   float64 `json:"amount, optional"`
	Fee     float64 `json:"fee, optional"`
	TradeNo string  `json:"transId, optional"`
	OrderNo string  `json:"orderId, optional"`
	Status  int     `json:"status, optional"`
}

type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
