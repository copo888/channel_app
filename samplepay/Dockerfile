FROM golang:alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOPROXY https://goproxy.cn,direct

WORKDIR /build/zero

ADD go.mod .
ADD go.sum .
RUN go mod download
COPY . .
COPY samplepay/etc /app/etc
RUN go build -ldflags="-s -w" -o /app/samplepay samplepay/samplepay.go


FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/samplepay /app/samplepay
COPY --from=builder /app/etc /app/etc

CMD ["./samplepay", "-f", "etc/samplepay.yaml"]
