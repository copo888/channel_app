FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/baisiangpay /app/baisiangpay
COPY baisiangpay/etc /app/etc

CMD ["./baisiangpay", "-f", "etc/pay.yaml"]