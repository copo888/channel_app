FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/wanjioupay /app/wanjioupay
COPY wanjioupay/etc /app/etc

CMD ["./wanjioupay", "-f", "etc/pay.yaml"]