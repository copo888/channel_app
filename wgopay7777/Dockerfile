FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wgopay7777 /app/wgopay7777
COPY wgopay7777/etc /app/etc

CMD ["./wgopay7777", "-f", "etc/pay.yaml"]