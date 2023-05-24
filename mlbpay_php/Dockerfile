FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mlbpay_php /app/mlbpay_php
COPY mlbpay_php/etc /app/etc

CMD ["./mlbpay_php", "-f", "etc/pay.yaml"]