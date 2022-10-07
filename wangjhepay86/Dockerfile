FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wangjhepay86 /app/wangjhepay86
COPY wangjhepay86/etc /app/etc

CMD ["./wangjhepay86", "-f", "etc/pay.yaml"]