package service

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/samplepay/internal/config"
	"github.com/copo888/channel_app/samplepay/internal/svc"
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

func DoCallTGSendURL(ctx context.Context, svcCtx *svc.ServiceContext, message string) error {
	span := trace.SpanFromContext(ctx)
	notifyUrl := fmt.Sprintf("%s:%d/telegram/notify", svcCtx.Config.TelegramSend.Host, svcCtx.Config.TelegramSend.Port)
	//notifyUrl := fmt.Sprintf("%s:%d/line/send", svcCtx.Config.LineSend.Host, svcCtx.Config.LineSend.Port)

	if _, err := gozzle.Post(notifyUrl).Timeout(25).Trace(span).JSON(message); err != nil {
		logx.WithContext(ctx).Errorf("报警通知失敗:%s", err.Error())
	}
	return nil
}
