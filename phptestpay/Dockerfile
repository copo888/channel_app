FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/phptestpay /app/phptestpay
COPY phptestpay/etc /app/etc

CMD ["./phptestpay", "-f", "etc/pay.yaml"]