FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/duotsaipay /app/duotsaipay
COPY duotsaipay/etc /app/etc

CMD ["./duotsaipay", "-f", "etc/pay.yaml"]