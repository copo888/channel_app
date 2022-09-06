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
	"strconv"
	"time"

	"github.com/copo888/channel_app/yibipay/internal/svc"
	"github.com/copo888/channel_app/yibipay/internal/types"

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
	aesKey := "qHp8VxRtzQ7HpBfE"
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

	//解密
	paramsDecode := utils.DePwdCode(req.Params, aesKey)
	logx.WithContext(l.ctx).Infof("解密后资料，paramsDecode: %s", paramsDecode)

	//反序列化
	chnResp := &channelResp{}
	if err = json.Unmarshal([]byte(paramsDecode), chnResp); err != nil {
		logx.WithContext(l.ctx).Errorf("反序列化失败: ", err)
	}
	orderStatus := "1"
	if chnResp.Code == "200" && chnResp.Data.OrderStatus == "1" {
		orderStatus = "20"
	}
	orderAmount, _ := strconv.ParseFloat(chnResp.Data.OrderPaidInAmount, 64)
	payCallBackBO := bo.PayCallBackBO{
		PayOrderNo:     chnResp.Data.DepositOrderId,
		ChannelOrderNo: chnResp.Data.TransactionId, // 渠道訂單號 (若无则填入->"CHN_" + orderNo)
		OrderStatus:    orderStatus,                // 若渠道只有成功会回调 固定 20:成功; 訂單狀態(1:处理中 20:成功 )
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

	return "success", nil
}

type channelResp struct {
	Code string `json:"code"` //200 代表成功,500 代表服务器内部错误,
	Data struct {
		CreatedAt           string `json:"createdAt,optional"`
		DepositOrderId      string `json:"depositOrderId,optional"`      //商户自己生成的唯一订单号
		OrderPaidInAmount   string `json:"orderPaidInAmount,optional"`   //用户实际付款金额
		OrderStatus         string `json:"orderStatus,optional"`         //1为确认成功，2为进行中，9确认失败
		RecommendDepositCny string `json:"recommendDepositCny,optional"` //推荐商户给会员充值人民币金额
		RequestAmount       string `json:"requestAmount,optional"`       //用户申请充值金额，页面显示的充值金额可能和实际充值金额不符
		RequestCurrency     string `json:"requestCurrency,optional"`
		SettlementAmount    string `json:"settlementAmount,optional"` //商户账户实际收款金额,商户账户实际的帐变金额,可能跟用户付款金额不同,	因为有汇率转换的汇差
		SettlementAssetType string `json:"settlementAssetType,optional"`
		TransactionId       string `json:"transactionId,optional"` //币汇记账凭证编号,商户账户到账的唯一凭证,请使用此Id进行幂等操作为会员上分
		TransactionType     string `json:"transactionType,optional"`
		UserCode            string `json:"userCode,optional"`
	} `json:"data,optional"`
	MerchantCode string `json:"merchantCode,optional"`
	Message      string `json:"message,optional"`
	Request      struct {
		MerchantCode string `json:"merchantCode"`
		MerchantId   string `json:"merchantId"`
		Timestamp    string `json:"timestamp"`
	} `json:"request,optional"`
	Timestamp string `json:"timestamp,optional"`
}
