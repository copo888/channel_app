FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/hueifutongpay /app/hueifutongpay
COPY hueifutongpay/etc /app/etc

CMD ["./hueifutongpay", "-f", "etc/pay.yaml"]