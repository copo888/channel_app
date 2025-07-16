FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/fpay /app/fpay
COPY fpay/etc /app/etc

CMD ["./fpay", "-f", "etc/pay.yaml"]