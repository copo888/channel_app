FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/z365pay /app/z365pay
COPY z365pay/etc /app/etc

CMD ["./z365pay", "-f", "etc/pay.yaml"]