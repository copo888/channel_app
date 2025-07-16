FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/indiapay /app/indiapay
COPY indiapay/etc /app/etc

CMD ["./indiapay", "-f", "etc/pay.yaml"]