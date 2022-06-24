FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/vp8pay /app/vp8pay
COPY vp8pay/etc /app/etc

CMD ["./vp8pay", "-f", "etc/pay.yaml"]