FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/z17pay /app/z17pay
COPY z17pay/etc /app/etc

CMD ["./z17pay", "-f", "etc/pay.yaml"]