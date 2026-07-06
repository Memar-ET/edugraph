package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv       string
	LogLevel     string
	Port         string
	AIServiceURL string
	CORSOrigins  []string
	AllowedHosts []string

	Postgres PostgresConfig
	Neo4j    Neo4jConfig
	Redis    RedisConfig
	JWT      JWTConfig
	OTEL     OTELConfig
	AWS      AWSConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	DB       string
	User     string
	Password string
	MaxConns int32
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DB)
}

type Neo4jConfig struct {
	URI             string
	User            string
	Password        string
	MaxConnPoolSize int
}

type RedisConfig struct {
	URL      string
	Password string
}

type JWTConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
}

type OTELConfig struct {
	Endpoint    string
	ServiceName string
}

type AWSConfig struct {
	Region           string
	CurriculumBucket string
	ExamBucket       string
	ReportsBucket    string
	AuditBucket      string
}

// Load reads configuration from environment variables, falling back to
// development-friendly defaults so the API can run without a .env file.
func Load() Config {
	return Config{
		AppEnv:       getenv("APP_ENV", "development"),
		LogLevel:     getenv("LOG_LEVEL", "info"),
		Port:         getenv("PORT", "8080"),
		AIServiceURL: getenv("AI_SERVICE_URL", "ai-service:8000"),
		CORSOrigins:  []string{getenv("FRONTEND_URL", "http://localhost:5173")},
		AllowedHosts: []string{},

		Postgres: PostgresConfig{
			Host:     getenv("POSTGRES_HOST", "postgres"),
			Port:     getenv("POSTGRES_PORT", "5432"),
			DB:       getenv("POSTGRES_DB", "edugraph"),
			User:     getenv("POSTGRES_USER", "edugraph"),
			Password: getenv("POSTGRES_PASSWORD", "devpass"),
			MaxConns: int32(getenvInt("POSTGRES_MAX_CONNS", 20)),
		},
		Neo4j: Neo4jConfig{
			URI:             getenv("NEO4J_URI", "bolt:neo4j:7687"),
			User:            getenv("NEO4J_USER", "neo4j"),
			Password:        getenv("NEO4J_PASSWORD", "devpass"),
			MaxConnPoolSize: getenvInt("NEO4J_MAX_CONN_POOL_SIZE", 50),
		},
		Redis: RedisConfig{
			URL:      getenv("REDIS_URL", "redis://redis:6379"),
			Password: getenv("REDIS_PASSWORD", ""),
		},
		JWT: JWTConfig{
			PrivateKeyPath: getenv("JWT_PRIVATE_KEY_PATH", "./secrets/jwt_private.pem"),
			PublicKeyPath:  getenv("JWT_PUBLIC_KEY_PATH", "./secrets/jwt_public.pem"),
			AccessTTL:      time.Duration(getenvInt("JWT_ACCESS_TTL_MINUTES", 15)) * time.Minute,
			RefreshTTL:     time.Duration(getenvInt("JWT_REFRESH_TTL_DAYS", 7)) * 24 * time.Hour,
		},
		OTEL: OTELConfig{
			Endpoint:    getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
			ServiceName: getenv("OTEL_SERVICE_NAME", "edugraph-api"),
		},
		AWS: AWSConfig{
			Region:           getenv("AWS_REGION", "af-south-1"),
			CurriculumBucket: getenv("AWS_S3_CURRICULUM_BUCKET", "edugraph-curriculum-docs"),
			ExamBucket:       getenv("AWS_S3_EXAM_BUCKET", "edugraph-exam-files"),
			ReportsBucket:    getenv("AWS_S3_REPORTS_BUCKET", "edugraph-reports"),
			AuditBucket:      getenv("AWS_S3_AUDIT_BUCKET", "edugraph-audit-logs"),
		},
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
