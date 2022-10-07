FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/tiansiapay /app/tiansiapay
COPY tiansiapay/etc /app/etc

CMD ["./tiansiapay", "-f", "etc/pay.yaml"]