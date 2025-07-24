FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/haihangpay /app/haihangpay
COPY haihangpay/etc /app/etc

CMD ["./haihangpay", "-f", "etc/pay.yaml"]