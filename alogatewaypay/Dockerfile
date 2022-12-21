FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/alogatewaypay /app/alogatewaypay
COPY alogatewaypay/etc /app/etc

CMD ["./alogatewaypay", "-f", "etc/pay.yaml"]