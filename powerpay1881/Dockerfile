FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/powerpay1881 /app/powerpay1881
COPY powerpay1881/etc /app/etc

CMD ["./powerpay1881", "-f", "etc/pay.yaml"]