FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wulinpay /app/wulinpay
COPY wulinpay/etc /app/etc

CMD ["./wulinpay", "-f", "etc/pay.yaml"]