FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/rcpay /app/rcpay
COPY rcpay/etc /app/etc

CMD ["./rcpay", "-f", "etc/pay.yaml"]