# syntax=docker/dockerfile:1
# Multi-stage build — produces a minimal School Box sync agent binary.
# The builder stage uses the same Go version as the main API.

FROM golang:1.22-alpine AS builder
WORKDIR /build

# Cache module downloads separately from source code so layer invalidation
# only hits the compile step when only source files change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" \
    -o /sync-agent ./cmd/sync-agent

# ─────────────────────────────
# Runtime image
# ─────────────────────────────
FROM alpine:3.19

# ca-certificates is needed for TLS connections to the cloud API.
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /sync-agent /usr/local/bin/sync-agent

# Health endpoint (also exposed in docker-compose HEALTHCHECK below).
EXPOSE 8081

USER nobody

ENTRYPOINT ["/usr/local/bin/sync-agent"]
