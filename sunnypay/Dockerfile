FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sunnypay /app/sunnypay
COPY sunnypay/etc /app/etc

CMD ["./sunnypay", "-f", "etc/pay.yaml"]