FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sinpay /app/sinpay
COPY sinpay/etc /app/etc

CMD ["./sinpay", "-f", "etc/pay.yaml"]