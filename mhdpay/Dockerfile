FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mhdpay /app/mhdpay
COPY mhdpay/etc /app/etc

CMD ["./mhdpay", "-f", "etc/pay.yaml"]