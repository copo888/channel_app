package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	model2 "github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/zhongshoupay/internal/payutils"
	"github.com/copo888/channel_app/zhongshoupay/internal/svc"
	"github.com/copo888/channel_app/zhongshoupay/internal/types"
	"github.com/gioco-play/gozzle"
	"go.opentelemetry.io/otel/trace"
	"net/url"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

type PayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	/** TODO: 測試code 要移除 **/
	amounts, err := strconv.ParseFloat(req.TransactionAmount, 64)
	receiverInfoJson, err := json.Marshal(types.ReceiverInfoVO{
		CardName: "王小明",
		CardNumber: "AAAA00001111",
		BankName: "BBB銀行",
		BankBranch: "AAA分行",
		Amount: amounts,
		Link: "is_test_url",
		Remark: "這是測試",
	})
	if strings.EqualFold(req.JumpType, "test") {
		// 測試反卡
		return &types.PayOrderResponse{
			PayPageType: "json",
			PayPageInfo: string(receiverInfoJson),
			Status:      "1", // 订单状态：状态 0处理中，1成功，2失败
			IsCheckOutMer: true,
		}, nil
	} else if strings.EqualFold(req.JumpType, "json") {
		// 測試返回json
		return &types.PayOrderResponse{
			PayPageType: "json",
			PayPageInfo: string(receiverInfoJson),
			Status:      "1", // 订单状态：状态 0处理中，1成功，2失败
		}, nil
	} else {
		// 正常測試
		return &types.PayOrderResponse{
			PayPageType: "url",
			PayPageInfo: "https://xuri.me/excelize/images/excelize.svg",
			Status:      "1", // 订单状态：状态 0处理中，1成功，2失败
		}, nil
	}
	/** TODO: 測試code 要移除 **/








	// 取得取道資訊
	channelModel := model2.NewChannel(l.svcCtx.MyDB)
	channel, err := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err.Error())
	}

	// 取值
	merchantId := channel.MerId
	merchantKey := channel.MerKey
	orderNo := req.OrderNo
	amount := req.TransactionAmount
	notifyUrl := channel.ApiUrl + "/api/pay-call-back"
	payType := req.PayType
	userId := req.UserId
	//ip := utils.GetRandomIp()

	// 組請求參數
	data := url.Values{}
	data.Set("partner", merchantId)
	data.Set("service", payType)
	data.Set("tradeNo", orderNo)
	data.Set("amount", amount)
	data.Set("notifyUrl", notifyUrl)
	data.Set("resultType", "json")

	if req.PayType == "YK" {
		if userId == "" {
			return nil, errorx.New(responsex.INVALID_USER_ID, err.Error())
		}
		data.Set("orderUserName", userId)
	}

	// 加簽
	sign := payutils.SortAndSignFromUrlValues(data, merchantKey)
	data.Set("sign", sign)

	// 請求渠道
	span := trace.SpanFromContext(l.ctx)
	res, err := gozzle.Post(channel.PayUrl).Timeout(10).Trace(span).Form(data)
	logx.Info(fmt.Sprintf("channel payOrder reply: url: %s, resp: %s ", channel.PayUrl, res))
	if err != nil {
		return nil, errorx.New(responsex.SERVICE_RESPONSE_ERROR, err.Error())
	}

	// 渠道回覆處理
	channelResp := struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg, optional"`
		Url     string `json:"url, optional"`
	}{}

	if err = res.DecodeJSON(&channelResp); err != nil {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err.Error())
	} else if !channelResp.Success {
		return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, channelResp.Msg)
	}

	resp = &types.PayOrderResponse{
		PayPageType: "url",
		PayPageInfo: channelResp.Url,
		Status:      "1", // 订单状态：状态 0处理中，1成功，2失败
	}

	return
}
