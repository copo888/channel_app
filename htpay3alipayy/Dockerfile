FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/htpay3alipayy /app/htpay3alipayy
COPY htpay3alipayy/etc /app/etc

CMD ["./htpay3alipayy", "-f", "etc/pay.yaml"]