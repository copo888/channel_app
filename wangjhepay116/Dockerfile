FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wangjhepay116 /app/wangjhepay116
COPY wangjhepay116/etc /app/etc

CMD ["./wangjhepay116", "-f", "etc/pay.yaml"]