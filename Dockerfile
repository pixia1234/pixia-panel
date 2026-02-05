FROM golang:1.22-bookworm AS builder
ARG TARGETARCH
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags "-s -w" -o /out/pixia-panel ./cmd/pixia-panel

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl tzdata && adduser -D -H -u 10001 app
WORKDIR /app

COPY --from=builder /out/pixia-panel /app/pixia-panel
COPY --from=builder /app/migrations /app/migrations

ENV PIXIA_HTTP_ADDR=:6365 \
    PIXIA_WS_PATH=/system-info \
    PIXIA_DB_PATH=/data/pixia.db

EXPOSE 6365
VOLUME ["/data"]
USER app

ENTRYPOINT ["/app/pixia-panel"]
