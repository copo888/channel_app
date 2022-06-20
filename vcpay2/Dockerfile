FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/vcpay2 /app/vcpay2
COPY vcpay2/etc /app/etc

CMD ["./vcpay2", "-f", "etc/pay.yaml"]