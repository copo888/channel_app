FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/u2pay /app/u2pay
COPY u2pay/etc /app/etc

CMD ["./u2pay", "-f", "etc/pay.yaml"]