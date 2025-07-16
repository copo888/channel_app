FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/youchengpay /app/youchengpay
COPY youchengpay/etc /app/etc

CMD ["./youchengpay", "-f", "etc/pay.yaml"]