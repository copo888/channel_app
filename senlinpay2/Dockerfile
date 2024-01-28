FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/senlinpay2 /app/senlinpay2
COPY senlinpay2/etc /app/etc

CMD ["./senlinpay2", "-f", "etc/pay.yaml"]