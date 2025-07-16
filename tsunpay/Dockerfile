FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/tsunpay /app/tsunpay
COPY tsunpay/etc /app/etc

CMD ["./tsunpay", "-f", "etc/pay.yaml"]