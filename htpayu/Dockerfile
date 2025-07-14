FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/htpayu /app/htpayu
COPY htpayu/etc /app/etc

CMD ["./htpayu", "-f", "etc/pay.yaml"]