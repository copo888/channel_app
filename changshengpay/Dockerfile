FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/changshengpay /app/changshengpay
COPY changshengpay/etc /app/etc

CMD ["./changshengpay", "-f", "etc/pay.yaml"]