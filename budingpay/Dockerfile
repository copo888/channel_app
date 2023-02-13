FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/budingpay /app/budingpay
COPY budingpay/etc /app/etc

CMD ["./budingpay", "-f", "etc/pay.yaml"]