FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/ddpay /app/ddpay
COPY ddpay/etc /app/etc

CMD ["./ddpay", "-f", "etc/pay.yaml"]