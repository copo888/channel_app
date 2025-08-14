FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/leileipay /app/leileipay
COPY leileipay/etc /app/etc

CMD ["./leileipay", "-f", "etc/pay.yaml"]