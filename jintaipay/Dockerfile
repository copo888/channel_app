FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jintaipay /app/jintaipay
COPY jintaipay/etc /app/etc

CMD ["./jintaipay", "-f", "etc/pay.yaml"]