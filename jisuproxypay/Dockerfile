FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jisuproxypay /app/jisuproxypay
COPY jisuproxypay/etc /app/etc

CMD ["./jisuproxypay", "-f", "etc/pay.yaml"]