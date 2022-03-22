package logic

import (
	"context"

	"github.com/copo888/channel_app/txpay/internal/svc"
	"github.com/copo888/channel_app/txpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxPayOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTxPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) TxPayOrderLogic {
	return TxPayOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TxPayOrderLogic) TxPayOrder(req *types.TxPayOrderRequest) (resp *types.OrderResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
