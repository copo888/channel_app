FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/lyubupay2 /app/lyubupay2
COPY lyubupay2/etc /app/etc

CMD ["./lyubupay2", "-f", "etc/pay.yaml"]