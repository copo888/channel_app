FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jinwangpay /app/jinwangpay
COPY jinwangpay/etc /app/etc

CMD ["./jinwangpay", "-f", "etc/pay.yaml"]