FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/htpay3_alipay /app/htpay3_alipay
COPY htpay3_alipay/etc /app/etc

CMD ["./htpay3_alipay", "-f", "etc/pay.yaml"]