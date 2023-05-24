FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/dpay /app/dpay
COPY dpay/etc /app/etc

CMD ["./dpay", "-f", "etc/pay.yaml"]