FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/shtpay /app/shtpay
COPY shtpay/etc /app/etc

CMD ["./shtpay", "-f", "etc/pay.yaml"]