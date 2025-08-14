FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mayipay /app/mayipay
COPY mayipay/etc /app/etc

CMD ["./mayipay", "-f", "etc/pay.yaml"]