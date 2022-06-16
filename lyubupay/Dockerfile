FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/lyubupay /app/lyubupay
COPY lyubupay/etc /app/etc

CMD ["./lyubupay", "-f", "etc/pay.yaml"]