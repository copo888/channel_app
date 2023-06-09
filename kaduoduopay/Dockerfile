FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/kaduoduopay /app/kaduoduopay
COPY kaduoduopay/etc /app/etc

CMD ["./kaduoduopay", "-f", "etc/pay.yaml"]