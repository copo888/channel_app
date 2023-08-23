FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kypay /app/kypay
COPY kypay/etc /app/etc

CMD ["./kypay", "-f", "etc/pay.yaml"]