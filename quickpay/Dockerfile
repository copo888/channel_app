FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/quickpay /app/quickpay
COPY quickpay/etc /app/etc

CMD ["./quickpay", "-f", "etc/pay.yaml"]