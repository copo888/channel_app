FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/testpayu /app/testpayu
COPY testpayu/etc /app/etc

CMD ["./testpayu", "-f", "etc/pay.yaml"]