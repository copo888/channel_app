FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/haojiehpay /app/haojiehpay
COPY haojiehpay/etc /app/etc

CMD ["./haojiehpay", "-f", "etc/pay.yaml"]