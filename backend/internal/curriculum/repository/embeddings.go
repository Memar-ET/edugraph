package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// UpsertCLOEmbedding stores (or replaces) the vector embedding for one CLO
// -- called by service.syncCLOEmbeddings after curriculum approval.
// embeddings.clo_embeddings.clo_code is UNIQUE (see db/migrations
// V011/V025), so this is a plain upsert, not a transaction: it's called
// once per CLO in a loop, and one failure shouldn't block the others.
//
// The vector is passed as a pgvector text literal ("[0.123,0.456,...]")
// cast to ::vector in SQL rather than via a typed driver value -- this
// repo has no pgvector Go driver dependency (see vectorLiteral below), so
// this avoids adding one just for a single INSERT.
func (r *Repository) UpsertCLOEmbedding(ctx context.Context, cloCode string, embedding []float32, modelVersion string) error {
	const q = `
		INSERT INTO embeddings.clo_embeddings (clo_code, embedding, model_ver)
		VALUES ($1, $2::vector, $3)
		ON CONFLICT (clo_code) DO UPDATE SET
			embedding  = EXCLUDED.embedding,
			model_ver  = EXCLUDED.model_ver,
			created_at = now()
	`
	if _, err := r.pool.Exec(ctx, q, cloCode, vectorLiteral(embedding), modelVersion); err != nil {
		return fmt.Errorf("upsert clo embedding %q: %w", cloCode, err)
	}
	return nil
}

// vectorLiteral formats a float32 slice as a pgvector text literal, e.g.
// []float32{0.1, -0.2} -> "[0.1,-0.2]". pgvector accepts this format cast
// to ::vector in a query, which is all a plain database/sql or pgx driver
// needs -- no pgvector-specific driver/extension required on the Go side.
func vectorLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
