package service

import (
	"context"
	"fmt"
	"github.com/copo888/channel_app/common/errorx"
	"github.com/copo888/channel_app/common/responsex"
	"github.com/copo888/channel_app/common/utils"
	"github.com/copo888/channel_app/jioufupay/internal/svc"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
)

func CallLineSendURL(ctx context.Context, svcCtx *svc.ServiceContext, message string) {
	go func() {
		DoCallLineSendURL(ctx, svcCtx, message)
	}()
}

func DoCallLineSendURL(ctx context.Context, svcCtx *svc.ServiceContext, message string) error {
	span := trace.SpanFromContext(ctx)
	notifyUrl := fmt.Sprintf("%s:%d/line/send", svcCtx.Config.LineSend.Host, svcCtx.Config.LineSend.Port)
	data := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}

	lineKey, errk := utils.MicroServiceEncrypt(svcCtx.Config.ApiKey.LineKey, svcCtx.Config.ApiKey.PublicKey)
	if errk != nil {
		logx.WithContext(ctx).Errorf("MicroServiceEncrypt: %s", errk.Error())
		return errorx.New(responsex.GENERAL_EXCEPTION, errk.Error())
	}

	res, errx := gozzle.Post(notifyUrl).Timeout(20).Trace(span).Header("authenticationLineKey", lineKey).JSON(data)
	if res != nil {
		logx.WithContext(ctx).Info("response Status:", res.Status())
		logx.WithContext(ctx).Info("response Body:", string(res.Body()))
	}
	if errx != nil {
		logx.WithContext(ctx).Errorf("call Channel cha: %s", errx.Error())
		return errorx.New(responsex.GENERAL_EXCEPTION, errx.Error())
	} else if res.Status() != 200 {
		return errorx.New(responsex.INVALID_STATUS_CODE, fmt.Sprintf("call channelApp httpStatus:%d", res.Status()))
	}

	return nil
}
