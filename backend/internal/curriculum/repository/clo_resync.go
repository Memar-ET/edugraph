package repository

import (
	"context"
	"fmt"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/internal/curriculum/dto"
)

// cloEdgeRow is one row fetched from the join of topic_clo_mappings + clos.
type cloEdgeRow struct {
	topicID     string
	code        string
	description string
}

// ResyncCLOsToNeo4j re-merges every (:Topic)-[:HAS_CLO]->(:CLO) edge in
// the graph. It fetches all rows from curriculum.topic_clo_mappings JOIN
// curriculum.clos and sends them to Neo4j in UNWIND batches of 500 so a
// large dataset doesn't exceed driver memory limits.
func (r *Repository) ResyncCLOsToNeo4j(ctx context.Context) (*dto.ResyncCLOsResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.topic_id::text, m.clo_code, COALESCE(c.description_en, '') AS description
		FROM curriculum.topic_clo_mappings m
		JOIN curriculum.clos c ON c.code = m.clo_code
	`)
	if err != nil {
		return nil, fmt.Errorf("query topic_clo_mappings: %w", err)
	}
	defer rows.Close()

	var edges []cloEdgeRow
	for rows.Next() {
		var e cloEdgeRow
		if err := rows.Scan(&e.topicID, &e.code, &e.description); err != nil {
			return nil, fmt.Errorf("scan clo edge: %w", err)
		}
		edges = append(edges, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clo edges: %w", err)
	}

	session := r.neo4j.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	const batchSize = 500
	synced, failed := 0, 0

	for i := 0; i < len(edges); i += batchSize {
		end := i + batchSize
		if end > len(edges) {
			end = len(edges)
		}
		batch := edges[i:end]

		params := make([]map[string]any, len(batch))
		for j, e := range batch {
			params[j] = map[string]any{
				"topicId":     e.topicID,
				"code":        e.code,
				"description": e.description,
			}
		}

		_, err := session.Run(ctx, `
			UNWIND $edges AS e
			MATCH (topic:Topic {id: e.topicId})
			MERGE (clo:CLO {code: e.code})
			SET clo.description = e.description
			MERGE (topic)-[:HAS_CLO]->(clo)
		`, map[string]any{"edges": params})
		if err != nil {
			failed += len(batch)
		} else {
			synced += len(batch)
		}
	}

	return &dto.ResyncCLOsResult{Synced: synced, Failed: failed}, nil
}
