FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/koreapay /app/koreapay
COPY koreapay/etc /app/etc

CMD ["./koreapay", "-f", "etc/pay.yaml"]