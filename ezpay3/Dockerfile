FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/ezpay3 /app/ezpay3
COPY ezpay3/etc /app/etc

CMD ["./ezpay3", "-f", "etc/pay.yaml"]