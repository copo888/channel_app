package logic

import (
	"context"
	"encoding/json"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/testpay/internal/svc"
	"github.com/copo888/channel_app/testpay/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"strconv"
	"strings"
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

	logx.WithContext(l.ctx).Infof("Enter PayOrder. channelName: %s, PayOrderRequest: %#v", l.svcCtx.Config.ProjectName, req)

	if strings.EqualFold(req.JumpType, "json") {
		transactionAmount, _ := strconv.ParseFloat( req.TransactionAmount, 64);

		// 返回json
		receiverInfoJson, err3 := json.Marshal(types.ReceiverInfoVO{
			CardName:  "王小銘",
			CardNumber: "11111111111111",
			BankName:   "工商银行",
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
	resp = &types.PayOrderResponse{
		PayPageType:    "url",
		PayPageInfo:    "https://docs.goldenf.vip/",
		ChannelOrderNo: "",
	}

	return
}
