package logic

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/lelifupay/internal/payutils"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"time"

	"github.com/copo888/channel_app/lelifupay/internal/svc"
	"github.com/copo888/channel_app/lelifupay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyPayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProxyPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) ProxyPayOrderQueryLogic {
	return ProxyPayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProxyPayOrderQueryLogic) ProxyPayOrderQuery(req *types.ProxyPayOrderQueryRequest) (resp *types.ProxyPayOrderQueryResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter ProxyPayOrderQuery. channelName: %s, ProxyPayOrderQueryRequest: %v", l.svcCtx.Config.ProjectName, req)
	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	logx.WithContext(l.ctx).Infof("代付订单channelName: %s, ChannelPayOrder: %v", channel.Name, req)
	timestamp := time.Now().Format("20060102150405")
	orderDate := time.Now().Format("20060102")

	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	data := url.Values{}
	data.Set("txnType", "00")
	data.Set("txnSubType", "50")
	data.Set("secpVer", "icp3-1.1")
	data.Set("secpMode", "perm")
	data.Set("macKeyId", channel.MerId)
	data.Set("merId", channel.MerId)
	data.Set("orderId", req.OrderNo)
	data.Set("orderDate", orderDate)
	data.Set("timeStamp", timestamp)

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, channel.MerKey)
	data.Set("mac", sign)

	logx.WithContext(l.ctx).Infof("代付查单请求地址:%s,代付請求參數:%#v", channel.ProxyPayQueryUrl, data)
	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	ChannelResp, ChnErr := gozzle.Post(channel.ProxyPayQueryUrl).Timeout(20).Trace(span).Form(data)

	if ChnErr != nil {
		logx.WithContext(l.ctx).Error("渠道返回錯誤: ", ChnErr.Error())
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, ChnErr.Error())
	} else if ChannelResp.Status() != 200 {
		logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
		return nil, errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("Error HTTP Status: %d", ChannelResp.Status()))
	}
	logx.WithContext(l.ctx).Infof("Status: %d  Body: %s", ChannelResp.Status(), string(ChannelResp.Body()))
	// 渠道回覆處理 [請依照渠道返回格式 自定義]
	channelQueryResp := struct {
		RespCode string `json:"respCode"`
		RespMsg string `json:"respMsg"`
		OrigRespCode string `json:"origRespCode"`
		OrigRespMsg string `json:"origRespMsg"`
		SecpVer string `json:"secpVer"`
		SecpMode string `json:"secpMode"`
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
		TxnStatus string `json:"txnStatus"`
		TxnStatusDesc string `json:"txnStatusDesc"`
		TimeStamp string `json:"timeStamp"`
		Mac string `json:"mac"`
	}{}

	if err3 := ChannelResp.DecodeJSON(&channelQueryResp); err3 != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
	} else if channelQueryResp.RespCode != "0000" {
		logx.WithContext(l.ctx).Errorf("代付查询渠道返回错误: %s: %s", channelQueryResp.RespCode, channelQueryResp.RespMsg)
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelQueryResp.RespMsg)
	}
	//0:待處理 1:處理中 20:成功 30:失敗 31:凍結
	var orderStatus = "1" //渠道回調狀態01---处理中 10---交易成功 20---交易失败 30---其他状态（需联系管理人员）
	if channelQueryResp.TxnStatus == "10" {
		orderStatus = "20"
	} else if channelQueryResp.TxnStatus == "10" {
		orderStatus = "2"
	} else if channelQueryResp.TxnStatus == "20" || channelQueryResp.TxnStatus == "30" {
		orderStatus = "3"
	}

	//組返回給BO 的代付返回物件
	return &types.ProxyPayOrderQueryResponse{
		Status: 1,
		//CallBackStatus: ""
		OrderStatus:      orderStatus,
		ChannelReplyDate: time.Now().Format("2006-01-02 15:04:05"),
		//ChannelCharge =
	}, nil
}
