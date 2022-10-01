FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/uzpay1881 /app/uzpay1881
COPY uzpay1881/etc /app/etc

CMD ["./uzpay1881", "-f", "etc/pay.yaml"]