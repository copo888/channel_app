FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mlbpay /app/mlbpay
COPY mlbpay/etc /app/etc

CMD ["./mlbpay", "-f", "etc/pay.yaml"]