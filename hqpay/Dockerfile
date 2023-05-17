FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/hqpay /app/hqpay
COPY hqpay/etc /app/etc

CMD ["./hqpay", "-f", "etc/pay.yaml"]