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
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"time"

	"github.com/copo888/channel_app/yunshengpay/internal/svc"
	"github.com/copo888/channel_app/yunshengpay/internal/types"

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

	logx.WithContext(l.ctx).Infof("Enter ProxyPayCallBack. channelName: %s, ProxyPayCallBackRequest: %#v", l.svcCtx.Config.ProjectName, req)

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

	response := utils.DePwdCode(req.Data, channel.MerKey)
	logx.WithContext(l.ctx).Infof("返回解密: %s", response)

	channelCallBackResp := struct {
		//Code    int     `json:"code, optional"`
		//Msg     string  `json:"msg, optional"`
		MerchId     string  `json:"merchantId, optional"`
		OrderNo     string  `json:"orderId, optional"`
		TradeNo     string  `json:"transId, optional"`
		Status      int     `json:"status, optional"` // (0, '成功') (2, '处理中'), (11, '取消'), (7, '撤单')
		Description string  `json:"description, optional"`
		Amount      float64 `json:"amount, optional"`
		Fee         float64 `json:"fee, optional"`
		Nonce       string  `json:"nonce, optional"`
	}{}

	if err = json.Unmarshal([]byte(response), &channelCallBackResp); err != nil {
		return "", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}

	//var orderAmount float64
	//if orderAmount, err = strconv.ParseFloat(req.Amount, 64); err != nil {
	//	return "fail", errorx.New(responsex.INVALID_SIGN)
	//}
	var status = "0" //渠道回調狀態(0:處理中1:成功2:失敗)
	if channelCallBackResp.Status == 0 {
		status = "1"
	} else if channelCallBackResp.Status == 11 || channelCallBackResp.Status == 7 {
		status = "2"
	}

	proxyPayCallBackBO := &bo.ProxyPayCallBackBO{
		ProxyPayOrderNo:     channelCallBackResp.OrderNo,
		ChannelOrderNo:      channelCallBackResp.TradeNo,
		ChannelResultAt:     time.Now().Format("20060102150405"),
		ChannelResultStatus: status,
		ChannelResultNote:   channelCallBackResp.Description,
		Amount:              channelCallBackResp.Amount,
		ChannelCharge:       0,
		UpdatedBy:           "",
	}

	// call boadmin callback api
	span := trace.SpanFromContext(l.ctx)
	payKey, errk := utils.MicroServiceEncrypt(l.svcCtx.Config.ApiKey.PayKey, l.svcCtx.Config.ApiKey.PublicKey)
	if errk != nil {
		resultJson, _ := json.Marshal(Result{
			Code: -1,
			Msg:  errk.Error(),
		})
		return string(resultJson), errk
		//return "fail", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	}
	//BoProxyRespVO := &vo.BoadminProxyRespVO{}
	url := fmt.Sprintf("%s:%d/dior/merchant-api/proxy-call-back", l.svcCtx.Config.Merchant.Host, l.svcCtx.Config.Merchant.Port)

	res, errx := gozzle.Post(url).Timeout(10).Trace(span).Header("authenticationPaykey", payKey).JSON(proxyPayCallBackBO)
	logx.Info("回调后资讯: ", res)
	if errx != nil {
		logx.WithContext(l.ctx).Error(errx.Error())
		resultJson, _ := json.Marshal(Result{
			Code: -1,
			Msg:  errx.Error(),
		})
		return string(resultJson), errx
		//return "fail", errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if res.Status() != 200 {
		resultJson, err := json.Marshal(Result{
			Code: -1,
			Msg:  fmt.Sprintf("status error:%d", res.Status()),
		})
		return string(resultJson), err
		//return "fail", errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("status:%d", res.Status()))
	}
	//else if errDecode:= res.DecodeJSON(BoProxyRespVO); errDecode!=nil {
	//   return "fail",errorx.New(responsex.DECODE_JSON_ERROR)
	//} else if BoProxyRespVO.Code != "000"{
	//	return "fail",errorx.New(BoProxyRespVO.Message)
	//}

	resultJson, err := json.Marshal(Result{
		Code: 0,
		Msg:  "成功",
	})
	return string(resultJson), nil
}
