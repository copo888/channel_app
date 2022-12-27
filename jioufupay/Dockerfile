FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jioufupay /app/jioufupay
COPY jioufupay/etc /app/etc

CMD ["./jioufupay", "-f", "etc/pay.yaml"]