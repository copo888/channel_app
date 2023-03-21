FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kumopaymaya /app/kumopaymaya
COPY kumopaymaya/etc /app/etc

CMD ["./kumopaymaya", "-f", "etc/pay.yaml"]