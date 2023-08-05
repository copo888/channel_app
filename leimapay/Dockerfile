FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/leimapay /app/leimapay
COPY leimapay/etc /app/etc

CMD ["./leimapay", "-f", "etc/pay.yaml"]