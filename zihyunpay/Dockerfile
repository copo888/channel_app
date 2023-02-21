FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/zihyunpay /app/zihyunpay
COPY zihyunpay/etc /app/etc

CMD ["./zihyunpay", "-f", "etc/pay.yaml"]