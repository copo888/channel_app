FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/typay /app/typay
COPY typay/etc /app/etc

CMD ["./typay", "-f", "etc/pay.yaml"]