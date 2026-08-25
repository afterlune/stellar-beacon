FROM golang:1.27.0-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY . .

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags '-w -s' -trimpath -a -o benetnasch

FROM alpine:3.24.1 AS final

#FROM scratch

ENV TZ="Asia/Shanghai"

WORKDIR /app

COPY --from=builder /build/benetnasch /app/

COPY ./resource /app/resource
COPY ./docs /app/docs

ENV GIN_MODE=release \
    PORT=7777

EXPOSE 7777

CMD ["/app/benetnasch"]
