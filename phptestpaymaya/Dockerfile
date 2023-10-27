FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/phptestpaymaya /app/phptestpaymaya
COPY phptestpaymaya/etc /app/etc

CMD ["./phptestpaymaya", "-f", "etc/pay.yaml"]