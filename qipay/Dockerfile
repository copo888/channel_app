FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/qipay /app/qipay
COPY qipay/etc /app/etc

CMD ["./qipay", "-f", "etc/pay.yaml"]