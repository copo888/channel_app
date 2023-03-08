FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/duomeipay /app/duomeipay
COPY duomeipay/etc /app/etc

CMD ["./duomeipay", "-f", "etc/pay.yaml"]