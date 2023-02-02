FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kumopay1881 /app/kumopay1881
COPY kumopay1881/etc /app/etc

CMD ["./kumopay1881", "-f", "etc/pay.yaml"]