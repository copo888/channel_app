FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/bitpayz2 /app/bitpayz2
COPY bitpayz2/etc /app/etc

CMD ["./bitpayz2", "-f", "etc/pay.yaml"]