FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/zzpay /app/zzpay
COPY zzpay/etc /app/etc

CMD ["./zzpay", "-f", "etc/pay.yaml"]