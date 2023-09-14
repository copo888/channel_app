FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/senlinpay /app/senlinpay
COPY senlinpay/etc /app/etc

CMD ["./senlinpay", "-f", "etc/pay.yaml"]