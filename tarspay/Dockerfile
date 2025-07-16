FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/tarspay /app/tarspay
COPY tarspay/etc /app/etc

CMD ["./tarspay", "-f", "etc/pay.yaml"]