FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jyuhueipay /app/jyuhueipay
COPY jyuhueipay/etc /app/etc

CMD ["./jyuhueipay", "-f", "etc/pay.yaml"]