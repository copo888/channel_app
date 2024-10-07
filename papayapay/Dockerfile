FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/papayapay /app/papayapay
COPY papayapay/etc /app/etc

CMD ["./papayapay", "-f", "etc/pay.yaml"]