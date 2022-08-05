FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/shandianpay /app/shandianpay
COPY shandianpay/etc /app/etc

CMD ["./shandianpay", "-f", "etc/pay.yaml"]