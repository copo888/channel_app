FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/nyypay /app/nyypay
COPY nyypay/etc /app/etc

CMD ["./nyypay", "-f", "etc/pay.yaml"]