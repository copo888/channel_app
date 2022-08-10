FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/yunshengpay /app/yunshengpay
COPY yunshengpay/etc /app/etc

CMD ["./yunshengpay", "-f", "etc/pay.yaml"]