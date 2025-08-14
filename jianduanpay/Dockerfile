FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jianduanpay /app/jianduanpay
COPY jianduanpay/etc /app/etc

CMD ["./jianduanpay", "-f", "etc/pay.yaml"]