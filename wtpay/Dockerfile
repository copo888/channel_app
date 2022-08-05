FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wtpay /app/wtpay
COPY wtpay/etc /app/etc

CMD ["./wtpay", "-f", "etc/pay.yaml"]