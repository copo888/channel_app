FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/mlbproxypay2 /app/mlbproxypay2
COPY mlbproxypay2/etc /app/etc

CMD ["./mlbproxypay2", "-f", "etc/pay.yaml"]