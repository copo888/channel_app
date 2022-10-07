FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/uzpay7777 /app/uzpay7777
COPY uzpay7777/etc /app/etc

CMD ["./uzpay7777", "-f", "etc/pay.yaml"]