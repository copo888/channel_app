FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/miduoduopay /app/miduoduopay
COPY miduoduopay/etc /app/etc

CMD ["./miduoduopay", "-f", "etc/pay.yaml"]