# ===== BUILD STAGE =====
FROM golang:1.27-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o server ./cmd/server/main.go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o migrate ./cmd/migrate

# ===== RUNTIME STAGE =====
FROM debian:bookworm-slim

# curl is for the docker-compose healthcheck on /readyz.
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/server /server
# For running migrations on their own: /migrate up|down|status
COPY --from=builder /app/migrate /migrate

EXPOSE 8083

CMD ["/server"]
