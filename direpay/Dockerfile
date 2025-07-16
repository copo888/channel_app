FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/direpay /app/direpay
COPY direpay/etc /app/etc

CMD ["./direpay", "-f", "etc/pay.yaml"]