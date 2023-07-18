FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/capitalpay /app/capitalpay
COPY capitalpay/etc /app/etc

CMD ["./capitalpay", "-f", "etc/pay.yaml"]