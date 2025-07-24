FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/infapay /app/infapay
COPY infapay/etc /app/etc

CMD ["./infapay", "-f", "etc/pay.yaml"]