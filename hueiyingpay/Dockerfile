FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/hueiyingpay /app/hueiyingpay
COPY hueiyingpay/etc /app/etc

CMD ["./hueiyingpay", "-f", "etc/pay.yaml"]