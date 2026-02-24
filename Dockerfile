# Multi-stage Dockerfile for RainLogs
# Produces two minimal images: `api` and `worker`

# ── Builder ────────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# Create a non-root user
RUN adduser -D -g '' appuser

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/rainlogs-api    ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/rainlogs-worker ./cmd/worker

# ── API image ───────────────────────────────────────────────────────────────
FROM scratch AS api

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/rainlogs-api /rainlogs-api

# Copy user/group info from builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Use non-root user
USER appuser:appuser

EXPOSE 8080
ENTRYPOINT ["/rainlogs-api"]

# ── Worker image ────────────────────────────────────────────────────────────
FROM scratch AS worker

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/rainlogs-worker /rainlogs-worker

# Copy user/group info from builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Use non-root user
USER appuser:appuser

ENTRYPOINT ["/rainlogs-worker"]
