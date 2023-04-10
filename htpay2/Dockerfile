FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/htpay2 /app/htpay2
COPY htpay2/etc /app/etc

CMD ["./htpay2", "-f", "etc/pay.yaml"]