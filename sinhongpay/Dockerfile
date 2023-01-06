FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sinhongpay /app/sinhongpay
COPY sinhongpay/etc /app/etc

CMD ["./sinhongpay", "-f", "etc/pay.yaml"]