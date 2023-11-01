FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/globalproxypay /app/globalproxypay
COPY globalproxypay/etc /app/etc

CMD ["./globalproxypay", "-f", "etc/pay.yaml"]