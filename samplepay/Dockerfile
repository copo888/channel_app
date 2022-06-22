FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/samplepay /app/samplepay
COPY samplepay/etc /app/etc

CMD ["./samplepay", "-f", "etc/pay.yaml"]