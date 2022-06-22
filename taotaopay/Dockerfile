FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/taotaopay /app/taotaopay
COPY taotaopay/etc /app/etc

CMD ["./taotaopay", "-f", "etc/pay.yaml"]