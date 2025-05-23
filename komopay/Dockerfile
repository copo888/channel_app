FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/komopay /app/komopay
COPY komopay/etc /app/etc

CMD ["./komopay", "-f", "etc/pay.yaml"]