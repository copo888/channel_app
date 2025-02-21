FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sunnypayys /app/sunnypayys
COPY sunnypayys/etc /app/etc

CMD ["./sunnypayys", "-f", "etc/pay.yaml"]