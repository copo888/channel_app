FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/dbinpay /app/dbinpay
COPY dbinpay/etc /app/etc

CMD ["./dbinpay", "-f", "etc/pay.yaml"]