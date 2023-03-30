FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/easypay /app/easypay
COPY easypay/etc /app/etc

CMD ["./easypay", "-f", "etc/pay.yaml"]