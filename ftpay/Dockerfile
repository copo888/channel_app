FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/ftpay /app/ftpay
COPY ftpay/etc /app/etc

CMD ["./ftpay", "-f", "etc/pay.yaml"]