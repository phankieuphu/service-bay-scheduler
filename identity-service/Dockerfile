# ===== BUILD STAGE =====
# Runs on the build host's own architecture and cross-compiles for the target
# one, so Apple Silicon and amd64 both get a native image without emulation.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

WORKDIR /app

# Dependencies first, so a source-only change reuses this layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/server ./cmd/migrate

# ===== RUNTIME STAGE =====
# busybox wget serves the docker-compose healthcheck on /readyz.
FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app && adduser -S -G app -H app

COPY --from=builder /out/server /server
# For running migrations on their own: /migrate up|down|status
COPY --from=builder /out/migrate /migrate

USER app
EXPOSE 8083

CMD ["/server"]
