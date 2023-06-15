FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/htpay3 /app/htpay3
COPY htpay3/etc /app/etc

CMD ["./htpay3", "-f", "etc/pay.yaml"]