FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wanlisinpay /app/wanlisinpay
COPY wanlisinpay/etc /app/etc

CMD ["./wanlisinpay", "-f", "etc/pay.yaml"]