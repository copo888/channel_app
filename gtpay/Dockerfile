FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/gtpay /app/gtpay
COPY gtpay/etc /app/etc

CMD ["./gtpay", "-f", "etc/pay.yaml"]