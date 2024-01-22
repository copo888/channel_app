FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/chiangshengpay /app/chiangshengpay
COPY chiangshengpay/etc /app/etc

CMD ["./chiangshengpay", "-f", "etc/pay.yaml"]