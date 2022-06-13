FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/zhongshoupay /app/zhongshoupay
COPY zhongshoupay/etc /app/etc

CMD ["./zhongshoupay", "-f", "etc/zhongshoupay.yaml"]