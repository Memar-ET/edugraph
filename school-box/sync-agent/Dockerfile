# ── Build stage ───────────────────────────────────────────────
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /app/bin/sync-agent ./cmd

# ── Final stage (distroless) ─────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/bin/sync-agent /app/sync-agent
EXPOSE 9090
ENTRYPOINT ["/app/sync-agent"]
