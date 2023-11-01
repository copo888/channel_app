FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kumoproxypay /app/kumoproxypay
COPY kumoproxypay/etc /app/etc

CMD ["./kumoproxypay", "-f", "etc/pay.yaml"]