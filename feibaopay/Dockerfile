FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/feibaopay /app/feibaopay
COPY feibaopay/etc /app/etc

CMD ["./feibaopay", "-f", "etc/pay.yaml"]