FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/bitpayz /app/bitpayz
COPY bitpayz/etc /app/etc

CMD ["./bitpayz", "-f", "etc/pay.yaml"]