FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/leimapay2 /app/leimapay2
COPY leimapay2/etc /app/etc

CMD ["./leimapay2", "-f", "etc/pay.yaml"]