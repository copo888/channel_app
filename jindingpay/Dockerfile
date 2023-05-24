FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jindingpay /app/jindingpay
COPY jindingpay/etc /app/etc

CMD ["./jindingpay", "-f", "etc/pay.yaml"]