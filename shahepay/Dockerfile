FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/shahepay /app/shahepay
COPY shahepay/etc /app/etc

CMD ["./shahepay", "-f", "etc/pay.yaml"]