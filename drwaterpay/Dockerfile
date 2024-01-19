FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/drwaterpay /app/drwaterpay
COPY drwaterpay/etc /app/etc

CMD ["./drwaterpay", "-f", "etc/pay.yaml"]