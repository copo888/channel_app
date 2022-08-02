FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sandianlingpay /app/sandianlingpay
COPY sandianlingpay/etc /app/etc

CMD ["./sandianlingpay", "-f", "etc/pay.yaml"]