package logic

import (
	"context"
	"github.com/copo888/channel_app/txpay/rpc/internal/svc"
	"github.com/copo888/channel_app/txpay/rpc/txpay"

	"github.com/zeromicro/go-zero/core/logx"
)

type TxPayOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTxPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TxPayOrderLogic {
	return &TxPayOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TxPayOrderLogic) TxPayOrder(in *txpay.TxPayOrderRequest) (*txpay.TxPayOrderResponse, error) {

	//endpoint :=  "http://example.com/submit_form.php"




	return &txpay.TxPayOrderResponse{}, nil
}
