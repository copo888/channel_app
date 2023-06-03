FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/newglobalpay1 /app/newglobalpay1
COPY newglobalpay1/etc /app/etc

CMD ["./newglobalpay1", "-f", "etc/pay.yaml"]