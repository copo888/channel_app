FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY /bin/rcpay_ue /app/rcpay_ue
COPY rcpay_ue/etc /app/etc

CMD ["./rcpay_ue", "-f", "etc/pay.yaml"]