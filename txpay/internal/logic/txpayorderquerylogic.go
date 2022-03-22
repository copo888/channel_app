package logic

import (
	"context"

	"github.com/copo888/channel_app/txpay/internal/svc"
	"github.com/copo888/channel_app/txpay/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxPayOrderQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTxPayOrderQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) TxPayOrderQueryLogic {
	return TxPayOrderQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TxPayOrderQueryLogic) TxPayOrderQuery(req *types.TxPayOrderQueryRequest) (resp *types.OrderResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
