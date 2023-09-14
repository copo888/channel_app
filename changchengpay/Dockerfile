FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/changchengpay /app/changchengpay
COPY changchengpay/etc /app/etc

CMD ["./changchengpay", "-f", "etc/pay.yaml"]