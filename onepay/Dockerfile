FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/onepay /app/onepay
COPY onepay/etc /app/etc

CMD ["./onepay", "-f", "etc/pay.yaml"]