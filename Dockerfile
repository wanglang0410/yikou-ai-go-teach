FROM golang:1.24-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server main.go

FROM alpine:3.21

WORKDIR /src

RUN apk add --no-cache ca-certificates curl tzdata \
    && addgroup -S app \
    && adduser -S app -G app

COPY --from=builder /server ./server
COPY go.mod go.sum ./
COPY config ./config
COPY prompt ./prompt
COPY docs ./docs

RUN printf '%s\n' \
    '#!/bin/sh' \
    'set -e' \
    'APP_ENV="${APP_ENV:-docker}"' \
    'CONFIG_FILE="/src/config/config-${APP_ENV}.yml"' \
    'if [ -n "$AI_API_KEY" ]; then sed -i "s|api-key:.*|api-key: ${AI_API_KEY}|" "$CONFIG_FILE"; fi' \
    'if [ -n "$MYSQL_PASSWORD" ]; then sed -i "s|password:.*|password: ${MYSQL_PASSWORD}|" "$CONFIG_FILE"; fi' \
    'exec ./server -env=${APP_ENV}' \
    > ./entrypoint.sh \
    && chmod +x ./entrypoint.sh \
    && mkdir -p tmp/code_output \
    && chown -R app:app /src

USER app

EXPOSE 8123

HEALTHCHECK --interval=10s --timeout=5s --retries=10 --start-period=20s \
  CMD curl -fsS http://127.0.0.1:8123/api/ping || exit 1

ENTRYPOINT ["./entrypoint.sh"]
