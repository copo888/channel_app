FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wanmahuipay /app/wanmahuipay
COPY wanmahuipay/etc /app/etc

CMD ["./wanmahuipay", "-f", "etc/pay.yaml"]