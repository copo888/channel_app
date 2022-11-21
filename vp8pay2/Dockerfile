FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/vp8pay2 /app/vp8pay2
COPY vp8pay2/etc /app/etc

CMD ["./vp8pay2", "-f", "etc/pay.yaml"]