FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/newglobalpay1111 /app/newglobalpay1111
COPY newglobalpay1111/etc /app/etc

CMD ["./newglobalpay1111", "-f", "etc/pay.yaml"]