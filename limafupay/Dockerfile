FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/limafupay /app/limafupay
COPY limafupay/etc /app/etc

CMD ["./limafupay", "-f", "etc/pay.yaml"]