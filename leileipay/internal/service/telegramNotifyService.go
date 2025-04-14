package service

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/leileipay/internal/config"
	"github.com/copo888/channel_app/leileipay/internal/svc"
	"github.com/copo888/channel_app/leileipay/internal/types"
	"github.com/gioco-play/gozzle"
	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config      config.Config
	RedisClient *redis.Client
	MyDB        *gorm.DB
}

func CallTGSendURL(ctx context.Context, svcCtx *svc.ServiceContext, message *types.TelegramNotifyRequest) {
	go func() {
		DoCallTGSendURL(ctx, svcCtx, message)
	}()
}

func DoCallTGSendURL(ctx context.Context, svcCtx *svc.ServiceContext, message *types.TelegramNotifyRequest) error {
	span := trace.SpanFromContext(ctx)
	notifyUrl := fmt.Sprintf("%s:%d/telegram/notify", svcCtx.Config.TelegramSend.Host, svcCtx.Config.TelegramSend.Port)

	if _, err := gozzle.Post(notifyUrl).Timeout(25).Trace(span).JSON(message); err != nil {
		logx.WithContext(ctx).Errorf("报警通知失敗:%s", err.Error())
	}
	return nil
}
