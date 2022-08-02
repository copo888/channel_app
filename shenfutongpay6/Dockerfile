FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/shenfutongpay6 /app/shenfutongpay6
COPY shenfutongpay6/etc /app/etc

CMD ["./shenfutongpay6", "-f", "etc/pay.yaml"]