FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/ailefupay /app/ailefupay
COPY ailefupay/etc /app/etc

CMD ["./ailefupay", "-f", "etc/pay.yaml"]