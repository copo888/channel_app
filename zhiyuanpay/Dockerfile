FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/zhiyuanpay /app/zhiyuanpay
COPY zhiyuanpay/etc /app/etc

CMD ["./zhiyuanpay", "-f", "etc/pay.yaml"]