package logic

import (
	"context"

	"github.com/copo888/channel_app/txpay/rpc/internal/svc"
	"github.com/copo888/channel_app/txpay/rpc/txpay"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxPayOrderQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTxPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TxPayOrderQueryLogic {
	return &TxPayOrderQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TxPayOrderQueryLogic) TxPayOrderQuery(in *txpay.TxPayQueryOrderRequest) (*txpay.TxPayQueryOrderResponse, error) {
	// todo: add your logic here and delete this line

	return &txpay.TxPayQueryOrderResponse{}, nil
}
