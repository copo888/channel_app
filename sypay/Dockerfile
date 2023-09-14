FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sypay /app/sypay
COPY sypay/etc /app/etc

CMD ["./sypay", "-f", "etc/pay.yaml"]