FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/bcpay /app/bcpay
COPY bcpay/etc /app/etc

CMD ["./bcpay", "-f", "etc/pay.yaml"]