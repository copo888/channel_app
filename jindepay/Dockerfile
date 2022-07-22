FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jindepay /app/jindepay
COPY jindepay/etc /app/etc

CMD ["./jindepay", "-f", "etc/pay.yaml"]