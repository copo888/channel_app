FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sunnypay2 /app/sunnypay2
COPY sunnypay2/etc /app/etc

CMD ["./sunnypay2", "-f", "etc/pay.yaml"]