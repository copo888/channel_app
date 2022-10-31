FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jstpay2 /app/jstpay2
COPY jstpay2/etc /app/etc

CMD ["./jstpay2", "-f", "etc/pay.yaml"]