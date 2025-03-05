package logic

import (
	"context"
	"encoding/json"
	"github.com/copo888/channel_app/common/constants"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/model"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/typesX"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/testpay/internal/service"
	"github.com/copo888/channel_app/testpay/internal/svc"
	"github.com/copo888/channel_app/testpay/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
)

type PayOrderLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) PayOrderLogic {
	return PayOrderLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *PayOrderLogic) PayOrder(req *types.PayOrderRequest) (resp *types.PayOrderResponse, err error) {

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %#v", l.svcCtx.Config.ProjectName, req)

	// 取得取道資訊
	channelModel := model.NewChannel(l.svcCtx.MyDB)
	channel, err1 := channelModel.GetChannelByProjectName(l.svcCtx.Config.ProjectName)
	if err1 != nil {
		return nil, errorx.New(responsex.INVALID_PARAMETER, err1.Error())
	}

	//// 渠道狀態碼判斷
	//if req.BankCode != "1" {
	//	//寫入交易日志
	//	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
	//		MerchantNo: req.MerchantId,
	//		//MerchantOrderNo: req.OrderNo,
	//		OrderNo:          req.OrderNo,
	//		LogType:          constants.ERROR_REPLIED_FROM_CHANNEL,
	//		LogSource:        constants.API_ZF,
	//		Content:          "Failed",
	//		TraceId:          l.traceID,
	//		ChannelErrorCode: "400",
	//	}); err != nil {
	//		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	//	}
	//	return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, "http: 400, Failed")
	//}

	//寫入交易日志
	if err := utils.CreateTransactionLog(l.svcCtx.MyDB, &typesX.TransactionLogData{
		MerchantNo:      req.MerchantId,
		MerchantOrderNo: req.MerchantOrderNo,
		OrderNo:         req.OrderNo,
		ChannelCode:     channel.Code,
		LogType:         constants.DATA_REQUEST_CHANNEL,
		LogSource:       constants.API_ZF,
		Content:         "Success",
		TraceId:         l.traceID,
	}); err != nil {
		logx.WithContext(l.ctx).Errorf("写入交易日志错误:%s", err)
	}

	if strings.EqualFold(req.JumpType, "json") {
		transactionAmount, _ := strconv.ParseFloat(req.TransactionAmount, 64)

		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:   "王小銘",
			CardNumber: "11111111111111",
			BankName:   "工商银行AX",
			BankBranch: "工商银行XX",
			Amount:     transactionAmount,
			Link:       "",
			Remark:     "",
		})
		if err3 != nil {
			return nil, errorx.New(responsex.CHANNEL_REPLY_ERROR, err3.Error())
		}
		return &types.PayOrderResponse{
			PayPageType:    "json",
			PayPageInfo:    string(receiverInfoJson),
			ChannelOrderNo: "",
			IsCheckOutMer:  true, // 自組收銀台回傳 true
		}, nil
	}

	service.CallTGSendURL(l.ctx, l.svcCtx, &types.TelegramNotifyRequest{ChatID: l.svcCtx.Config.TelegramSend.ChatId, Message: "測試"})

	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    "https://docs.goldenf.vip/",
		ChannelOrderNo: "",
	}

	return
}
