FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jstpay3 /app/jstpay3
COPY jstpay3/etc /app/etc

CMD ["./jstpay3", "-f", "etc/pay.yaml"]