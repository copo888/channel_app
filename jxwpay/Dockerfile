FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jxwpay /app/jxwpay
COPY jxwpay/etc /app/etc

CMD ["./jxwpay", "-f", "etc/pay.yaml"]