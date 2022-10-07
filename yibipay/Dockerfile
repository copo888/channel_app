FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/yibipay /app/yibipay
COPY yibipay/etc /app/etc

CMD ["./yibipay", "-f", "etc/pay.yaml"]