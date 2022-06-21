FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sinshangpay /app/sinshangpay
COPY sinshangpay/etc /app/etc

CMD ["./sinshangpay", "-f", "etc/pay.yaml"]