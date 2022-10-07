FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/shenfutongpay7 /app/shenfutongpay7
COPY shenfutongpay7/etc /app/etc

CMD ["./shenfutongpay7", "-f", "etc/pay.yaml"]