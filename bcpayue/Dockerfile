FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/bcpayue /app/bcpayue
COPY bcpayue/etc /app/etc

CMD ["./bcpayue", "-f", "etc/pay.yaml"]