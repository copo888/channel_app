FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/tesihlapay /app/tesihlapay
COPY tesihlapay/etc /app/etc

CMD ["./tesihlapay", "-f", "etc/pay.yaml"]