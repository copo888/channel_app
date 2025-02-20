FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jsphppay /app/jsphppay
COPY jsphppay/etc /app/etc

CMD ["./jsphppay", "-f", "etc/pay.yaml"]