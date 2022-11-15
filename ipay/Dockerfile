FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/ipay /app/ipay
COPY ipay/etc /app/etc

CMD ["./ipay", "-f", "etc/pay.yaml"]