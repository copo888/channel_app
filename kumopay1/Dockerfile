FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kumopay1 /app/kumopay1
COPY kumopay1/etc /app/etc

CMD ["./kumopay1", "-f", "etc/pay.yaml"]