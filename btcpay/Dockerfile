FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/btcpay /app/btcpay
COPY btcpay/etc /app/etc

CMD ["./btcpay", "-f", "etc/pay.yaml"]