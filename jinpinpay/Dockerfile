FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jinpinpay /app/jinpinpay
COPY jinpinpay/etc /app/etc

CMD ["./jinpinpay", "-f", "etc/pay.yaml"]