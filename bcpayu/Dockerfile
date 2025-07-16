FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/bcpayu /app/bcpayu
COPY bcpayu/etc /app/etc

CMD ["./bcpayu", "-f", "etc/pay.yaml"]