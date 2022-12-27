FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wgopay88 /app/wgopay88
COPY wgopay88/etc /app/etc

CMD ["./wgopay88", "-f", "etc/pay.yaml"]