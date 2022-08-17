FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/testpay /app/testpay
COPY testpay/etc /app/etc

CMD ["./testpay", "-f", "etc/pay.yaml"]