FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kumopay1111 /app/kumopay1111
COPY kumopay1111/etc /app/etc

CMD ["./kumopay1111", "-f", "etc/pay.yaml"]