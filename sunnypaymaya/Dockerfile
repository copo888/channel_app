FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sunnypaymaya /app/sunnypaymaya
COPY sunnypaymaya/etc /app/etc

CMD ["./sunnypaymaya", "-f", "etc/pay.yaml"]