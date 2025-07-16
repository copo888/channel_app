FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/glpay /app/glpay
COPY glpay/etc /app/etc

CMD ["./glpay", "-f", "etc/pay.yaml"]