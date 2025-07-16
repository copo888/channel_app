FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jstpay4 /app/jstpay4
COPY jstpay4/etc /app/etc

CMD ["./jstpay4", "-f", "etc/pay.yaml"]