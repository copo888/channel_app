FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/ezpay2 /app/ezpay2
COPY ezpay2/etc /app/etc

CMD ["./ezpay2", "-f", "etc/pay.yaml"]