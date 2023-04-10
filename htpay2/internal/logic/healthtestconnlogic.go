package logic

import (
	"context"
	"go.opentelemetry.io/otel/trace"

	"github.com/copo888/channel_app/htpay2/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type HealthTestConnLogic struct {
	logx.Logger
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	traceID string
}

func NewHealthTestConnLogic(ctx context.Context, svcCtx *svc.ServiceContext) HealthTestConnLogic {
	return HealthTestConnLogic{
		Logger:  logx.WithContext(ctx),
		ctx:     ctx,
		svcCtx:  svcCtx,
		traceID: trace.SpanContextFromContext(ctx).TraceID().String(),
	}
}

func (l *HealthTestConnLogic) HealthTestConn() (resp string, err error) {

	resp = "success"

	return resp, nil
}
