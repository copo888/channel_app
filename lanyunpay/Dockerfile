FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/lanyunpay /app/lanyunpay
COPY lanyunpay/etc /app/etc

CMD ["./lanyunpay", "-f", "etc/pay.yaml"]