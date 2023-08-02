FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/lelifupay2 /app/lelifupay2
COPY lelifupay2/etc /app/etc

CMD ["./lelifupay2", "-f", "etc/pay.yaml"]