FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/luckypay /app/luckypay
COPY luckypay/etc /app/etc

CMD ["./luckypay", "-f", "etc/pay.yaml"]