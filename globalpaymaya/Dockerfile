FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/globalpaymaya /app/globalpaymaya
COPY globalpaymaya/etc /app/etc

CMD ["./globalpaymaya", "-f", "etc/pay.yaml"]