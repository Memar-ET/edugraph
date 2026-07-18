package neo4j

import (
	"context"
	"fmt"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	neo4jconfig "github.com/neo4j/neo4j-go-driver/v5/neo4j/config"

	"github.com/edugraph-ai/edugraph/pkg/config"
)

// NewDriver creates and verifies connectivity for a Neo4j driver.
func NewDriver(cfg config.Neo4jConfig) (neo4jdriver.DriverWithContext, error) {
	driver, err := neo4jdriver.NewDriverWithContext(
		cfg.URI,
		neo4jdriver.BasicAuth(cfg.User, cfg.Password, ""),
		func(c *neo4jconfig.Config) {
			if cfg.MaxConnPoolSize > 0 {
				c.MaxConnectionPoolSize = cfg.MaxConnPoolSize
			}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver: %w", err)
	}

	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		_ = driver.Close(context.Background())
		return nil, fmt.Errorf("verify neo4j connectivity: %w", err)
	}

	return driver, nil
}
