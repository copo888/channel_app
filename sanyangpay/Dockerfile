FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/sanyangpay /app/sanyangpay
COPY sanyangpay/etc /app/etc

CMD ["./sanyangpay", "-f", "etc/pay.yaml"]