FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mashangpay8888 /app/mashangpay8888
COPY mashangpay8888/etc /app/etc

CMD ["./mashangpay8888", "-f", "etc/pay.yaml"]