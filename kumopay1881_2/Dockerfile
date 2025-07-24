FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kumopay1881_2 /app/kumopay1881_2
COPY kumopay1881_2/etc /app/etc

CMD ["./kumopay1881_2", "-f", "etc/pay.yaml"]