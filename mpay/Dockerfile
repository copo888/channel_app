FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mpay /app/mpay
COPY mpay/etc /app/etc

CMD ["./mpay", "-f", "etc/pay.yaml"]