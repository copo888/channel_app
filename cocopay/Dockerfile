FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/cocopay /app/cocopay
COPY cocopay/etc /app/etc

CMD ["./cocopay", "-f", "etc/pay.yaml"]