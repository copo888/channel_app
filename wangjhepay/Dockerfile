FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wangjhepay /app/wangjhepay
COPY wangjhepay/etc /app/etc

CMD ["./wangjhepay", "-f", "etc/pay.yaml"]
