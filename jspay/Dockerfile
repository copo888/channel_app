FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/jspay /app/jspay
COPY jspay/etc /app/etc

CMD ["./jspay", "-f", "etc/pay.yaml"]