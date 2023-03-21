FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kumopayMaya /app/kumopayMaya
COPY kumopayMaya/etc /app/etc

CMD ["./kumopayMaya", "-f", "etc/pay.yaml"]