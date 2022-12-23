FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kumopay /app/kumopay
COPY kumopay/etc /app/etc

CMD ["./kumopay", "-f", "etc/pay.yaml"]