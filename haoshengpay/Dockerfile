FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/haoshengpay /app/haoshengpay
COPY haoshengpay/etc /app/etc

CMD ["./haoshengpay", "-f", "etc/pay.yaml"]