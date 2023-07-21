FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jinpinpay2 /app/jinpinpay2
COPY jinpinpay2/etc /app/etc

CMD ["./jinpinpay2", "-f", "etc/pay.yaml"]