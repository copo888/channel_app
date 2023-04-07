FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/uhpay /app/uhpay
COPY uhpay/etc /app/etc

CMD ["./uhpay", "-f", "etc/pay.yaml"]