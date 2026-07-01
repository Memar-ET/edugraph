module github.com/edugraph-ai/edugraph

go 1.22

require (
  github.com/gin-gonic/gin v1.10.0
  github.com/jackc/pgx/v5 v5.5.5
  github.com/neo4j/neo4j-go-driver/v5 v5.20.0
  github.com/redis/rueidis v1.0.35
  github.com/golang-jwt/jwt/v5 v5.2.1
  github.com/go-playground/validator/v10 v10.20.0
  go.uber.org/zap v1.27.0
  github.com/prometheus/client_golang v1.19.0
  github.com/gorilla/websocket v1.5.1
  github.com/sony/gobreaker v0.5.0
  golang.org/x/sync v0.7.0
  github.com/google/uuid v1.6.0
  github.com/pgvector/pgvector-go v0.1.1
  golang.org/x/crypto v0.23.0
  github.com/jackc/pglogrepl v0.0.0-20240307033717-828fbfe908e9
  github.com/stretchr/testify v1.9.0
  github.com/testcontainers/testcontainers-go v0.30.0
  go.opentelemetry.io/otel v1.26.0
  go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.26.0
)
