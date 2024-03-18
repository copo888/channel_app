FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/ezpay /app/ezpay
COPY ezpay/etc /app/etc

CMD ["./ezpay", "-f", "etc/pay.yaml"]