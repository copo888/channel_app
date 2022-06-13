FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/lepay /app/lepay
COPY lepay/etc /app/etc

CMD ["./lepay", "-f", "etc/pay.yaml"]
