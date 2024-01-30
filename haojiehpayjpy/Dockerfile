FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/haojiehpayjpy /app/haojiehpayjpy
COPY haojiehpayjpy/etc /app/etc

CMD ["./haojiehpayjpy", "-f", "etc/pay.yaml"]