FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wangjhepay87 /app/wangjhepay87
COPY wangjhepay87/etc /app/etc

CMD ["./wangjhepay87", "-f", "etc/pay.yaml"]