FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/globalpay /app/globalpay
COPY globalpay/etc /app/etc

CMD ["./globalpay", "-f", "etc/pay.yaml"]