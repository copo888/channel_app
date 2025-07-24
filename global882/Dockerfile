FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/global882 /app/global882
COPY global882/etc /app/etc

CMD ["./global882", "-f", "etc/pay.yaml"]