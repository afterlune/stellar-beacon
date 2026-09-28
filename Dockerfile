FROM golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://proxy.golang.org,direct

WORKDIR /build

COPY . .

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags '-w -s' -trimpath -a -o stellar-beacon ./cmd/stellar-beacon

FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS final

#FROM scratch

ENV TZ="Asia/Shanghai"

WORKDIR /app

COPY --from=builder /build/stellar-beacon /app/

COPY ./resources /app/resources
COPY ./deploy/config /app/config
COPY ./docs /app/docs

RUN apk upgrade --no-cache \
    && apk add --no-cache curl tzdata \
    && addgroup -S -g 10001 app \
    && adduser -S -u 10001 -G app app \
    && mkdir -p /app/resources/log /var/lib/stellar-beacon/keys \
    && chown -R 10001:10001 /app/resources/log /var/lib/stellar-beacon \
    && chmod 0700 /var/lib/stellar-beacon/keys

ENV GIN_MODE=release \
    PORT=7777 \
    STELLAR_BEACON_RESOURCE_DIR=/app/resources \
    STELLAR_BEACON_CONFIG_DIR=/app/config

EXPOSE 7777

USER 10001:10001

CMD ["/app/stellar-beacon"]
