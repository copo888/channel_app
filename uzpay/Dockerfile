FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/uzpay /app/uzpay
COPY uzpay/etc /app/etc

CMD ["./uzpay", "-f", "etc/pay.yaml"]