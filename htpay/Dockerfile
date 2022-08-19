FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/htpay /app/htpay
COPY htpay/etc /app/etc

CMD ["./htpay", "-f", "etc/pay.yaml"]