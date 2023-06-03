FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/newglobalpay1881 /app/newglobalpay1881
COPY newglobalpay1881/etc /app/etc

CMD ["./newglobalpay1881", "-f", "etc/pay.yaml"]