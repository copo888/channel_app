FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/global88 /app/global88
COPY global88/etc /app/etc

CMD ["./global88", "-f", "etc/pay.yaml"]