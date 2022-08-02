FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/stpay /app/stpay
COPY stpay/etc /app/etc

CMD ["./stpay", "-f", "etc/pay.yaml"]