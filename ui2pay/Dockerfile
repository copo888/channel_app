FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/ui2pay /app/ui2pay
COPY ui2pay/etc /app/etc

CMD ["./ui2pay", "-f", "etc/pay.yaml"]