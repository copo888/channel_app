FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/newglobalpay777 /app/newglobalpay777
COPY newglobalpay777/etc /app/etc

CMD ["./newglobalpay777", "-f", "etc/pay.yaml"]