FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/quickpay_va /app/quickpay_va
COPY quickpay_va/etc /app/etc

CMD ["./quickpay_va", "-f", "etc/pay.yaml"]