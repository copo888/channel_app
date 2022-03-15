package logic

import (
	"context"

	"github.com/copo888/channel_app/txpay/rpc/internal/svc"
	"github.com/copo888/channel_app/txpay/rpc/txpay"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxProxyPayQueryBalanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTxProxyPayQueryBalanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TxProxyPayQueryBalanceLogic {
	return &TxProxyPayQueryBalanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TxProxyPayQueryBalanceLogic) TxProxyPayQueryBalance(in *txpay.TxProxyPayQueryInternalBalanceRequest) (*txpay.TxProxyPayQueryInternalBalanceResponse, error) {
	// todo: add your logic here and delete this line

	return &txpay.TxProxyPayQueryInternalBalanceResponse{}, nil
}
