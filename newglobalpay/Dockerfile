FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/newglobalpay /app/newglobalpay
COPY newglobalpay/etc /app/etc

CMD ["./newglobalpay", "-f", "etc/pay.yaml"]