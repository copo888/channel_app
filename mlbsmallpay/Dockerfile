FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mlbsmallpay /app/mlbsmallpay
COPY mlbsmallpay/etc /app/etc

CMD ["./mlbsmallpay", "-f", "etc/pay.yaml"]