FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/yuehhaopay /app/yuehhaopay
COPY yuehhaopay/etc /app/etc

CMD ["./yuehhaopay", "-f", "etc/pay.yaml"]