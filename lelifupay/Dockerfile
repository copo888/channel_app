FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/lelifupay /app/lelifupay
COPY lelifupay/etc /app/etc

CMD ["./lelifupay", "-f", "etc/pay.yaml"]