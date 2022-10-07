FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mashangpay /app/mashangpay
COPY mashangpay/etc /app/etc

CMD ["./mashangpay", "-f", "etc/pay.yaml"]