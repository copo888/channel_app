FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/huanyupay /app/huanyupay
COPY huanyupay/etc /app/etc

CMD ["./huanyupay", "-f", "etc/pay.yaml"]