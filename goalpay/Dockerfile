FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/goalpay /app/goalpay
COPY goalpay/etc /app/etc

CMD ["./goalpay", "-f", "etc/pay.yaml"]