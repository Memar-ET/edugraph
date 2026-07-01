# ── Build stage ───────────────────────────────────────────────
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install dependencies first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /app/bin/api ./cmd/api

# ── Final stage (distroless) ─────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/bin/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]
