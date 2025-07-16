FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/shuangsipay /app/shuangsipay
COPY shuangsipay/etc /app/etc

CMD ["./shuangsipay", "-f", "etc/pay.yaml"]