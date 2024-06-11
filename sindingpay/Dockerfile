FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sindingpay /app/sindingpay
COPY sindingpay/etc /app/etc

CMD ["./sindingpay", "-f", "etc/pay.yaml"]