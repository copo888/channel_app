FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wgopay /app/wgopay
COPY wgopay/etc /app/etc

CMD ["./wgopay", "-f", "etc/pay.yaml"]