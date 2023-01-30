FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/laikepay /app/laikepay
COPY laikepay/etc /app/etc

CMD ["./laikepay", "-f", "etc/pay.yaml"]