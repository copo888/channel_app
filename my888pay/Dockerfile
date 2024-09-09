FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/my888pay /app/my888pay
COPY my888pay/etc /app/etc

CMD ["./my888pay", "-f", "etc/pay.yaml"]