package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/lelifupay2/internal/payutils"
	"github.com/copo888/channel_app/lelifupay2/internal/svc"
	"github.com/copo888/channel_app/lelifupay2/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderQueryLogic {
	return PayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderQueryLogic) PayOrderQuery(req *types.PayOrderQueryRequest) (resp *types.PayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrderQuery. channelName: %s, PayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	var channel typesX.ChannelData
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	if channel, err = channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName); err != nil {
		return
	}
	timestamp := time.Now().Format("20060102150405")
	orderDate := time.Now().Format("20060102")

	// 組請求參數
	data := url.Values{}
	data.Set("txnType", "00")
	data.Set("txnSubType", "10")
	data.Set("secpVer", "icp3-1.1")
	data.Set("secpMode", "perm")
	data.Set("macKeyId", channel.MerId)
	data.Set("merId", channel.MerId)
	if req.OrderNo != "" {
		data.Set("orderId", req.OrderNo)
	}
	data.Set("orderDate", orderDate)
	data.Set("timeStamp", timestamp)

	// 組請求參數 FOR JSON
	//data := struct {
	//	merchId  string
	//	orderId  string
	//	time     string
	//	signType string
	//	sign     string
	//}{
	//	merchId:  channel.MerId,
	//	orderId:  req.OrderNo,
	//	time:     timestamp,
	//	signType: "MD5",
	//}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("mac", sign)

	// 加簽 JSON
	//sign := payutils.SortAndSignFromObj(data, channel.MerKey)
	//data.sign = sign

	// 請求渠道
	logx.WithContext(l.ctx).Infof("支付查詢请求地址:%s,支付請求參數:%v", channel.PayQueryUrl, data)

	span := trace.SpanFromContext(l.ctx)
	res, chnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).Form(data)
	//res, ChnErr := gozzle.Post(channel.PayQueryUrl).Timeout(20).Trace(span).JSON(data)

	if chnErr != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_DATA_ERROR, err.Error())
	} else if res.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", res.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", res.Status(), string(res.Body()))

	// 渠道回覆處理
	channelResp := struct {
		RespCode string `json:"respCode"`
		RespMsg  string `json:"respMsg"`
		OrigRespCode string `json:"origRespCode"`
		OrigRespMsg string `json:"origRespMsg"`
		SecpMode string `json:"secpMode"`
		SecpVer string `json:"secpVer"`
		MacKeyId string `json:"macKeyId"`
		OrderDate string `json:"orderDate"`
		OrderTime string `json:"orderTime"`
		FinishDate string `json:"finishDate"`
		FinishTime string `json:"finishTime"`
		MerId string `json:"merId"`
		ExtInfo string `json:"extInfo"`
		OrderId string `json:"orderId"`
		TxnId string `json:"txnId"`
		TxnAmt string `json:"txnAmt"`
		CurrencyCode string `json:"currencyCode"`
		TxnStatus string `json:"txnStatus"` //01---处理中 10---交易成功 20---交易失败 30---其他状态（需联系管理人员）
		TxnStatusDesc string `json:"txnStatusDesc"`
		TimeStamp string `json:"timeStamp"`
		Mac string `json:"mac"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.GENERAL_EXCEPTION, err.Error())
	} else if channelResp.RespCode != "0000" {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.RespMsg)
	}

	orderAmount := utils.FloatDiv(channelResp.TxnAmt, "100")

	//orderAmount, errParse := strconv.ParseFloat(channelResp.txnAmt, 64)
	//if errParse != nil {
	//	return nil, errorx.New(responsex.GENERAL_EXCEPTION, errParse.Error())
	//}

	orderStatus := "0"
	if channelResp.TxnStatus == "10" {
		orderStatus = "1"
	}else if channelResp.TxnStatus == "20" || channelResp.TxnStatus == "30" {
		orderStatus = "2"
	}

	resp = &types.PayOrderQueryResponse{
		OrderAmount: orderAmount,
		OrderStatus: orderStatus, //订单状态: 状态 0处理中，1成功，2失败
	}

	return
}
