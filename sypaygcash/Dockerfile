FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sypaygcash /app/sypaygcash
COPY sypaygcash/etc /app/etc

CMD ["./sypaygcash", "-f", "etc/pay.yaml"]