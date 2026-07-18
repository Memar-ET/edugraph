package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ErrNotFound is shared across every repository method in this package
// (exam_upload.go, exam_validate.go, exam_submit.go, ...).
var ErrNotFound = errors.New("not found")

type Repository struct {
	pool  *pgxpool.Pool
	neo4j neo4jdriver.DriverWithContext
}

func New(pool *pgxpool.Pool, neo4j neo4jdriver.DriverWithContext) *Repository {
	return &Repository{pool: pool, neo4j: neo4j}
}
