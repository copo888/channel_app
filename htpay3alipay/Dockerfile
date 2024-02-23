FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/htpay3alipay /app/htpay3alipay
COPY htpay3alipay/etc /app/etc

CMD ["./htpay3alipay", "-f", "etc/pay.yaml"]