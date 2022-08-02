FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/liepay /app/liepay
COPY liepay/etc /app/etc

CMD ["./liepay", "-f", "etc/pay.yaml"]