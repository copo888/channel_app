FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jstpay /app/jstpay
COPY jstpay/etc /app/etc

CMD ["./jstpay", "-f", "etc/pay.yaml"]