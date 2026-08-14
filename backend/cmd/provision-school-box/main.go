// Command provision-school-box registers one School Box's sync-agent
// credential against the Central Cloud database and prints the plaintext
// secret exactly once. There is no HTTP endpoint for this deliberately --
// provisioning a physical box is a rare, manual, operator-driven action
// (the same "fill in .env.school-box by hand" model install.sh already
// assumes), not something that needs its own admin UI yet. Run this once
// per box, before its first sync-agent boot, and paste the printed secret
// into that box's .env.school-box as SCHOOL_BOX_SECRET.
//
// See V030__sync_device_credentials.sql and
// internal/sync/handler/device_auth.go for why this exists: /api/v1/sync
// /push and /pull used to accept any caller with no authentication at
// all (checklist 10.1 finding).
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"github.com/edugraph-ai/edugraph/pkg/config"
	"github.com/edugraph-ai/edugraph/pkg/crypto"
	"github.com/edugraph-ai/edugraph/pkg/database/postgres"
)

func main() {
	deviceID := flag.String("device-id", "", "SCHOOL_BOX_ID of the box being provisioned (required)")
	schoolID := flag.String("school-id", "", "UUID of the school this box belongs to, from public.schools (required)")
	flag.Parse()

	if *deviceID == "" || *schoolID == "" {
		log.Fatal("both -device-id and -school-id are required")
	}

	secret, err := generateSecret()
	if err != nil {
		log.Fatalf("generate secret: %v", err)
	}
	hash, err := crypto.HashPassword(secret)
	if err != nil {
		log.Fatalf("hash secret: %v", err)
	}

	cfg := config.Load()
	pool, err := postgres.NewPool(cfg.Postgres)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	const q = `INSERT INTO sync.device_credentials (device_id, school_id, secret_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (device_id) DO UPDATE SET school_id = $2, secret_hash = $3, revoked_at = NULL`
	if _, err := pool.Exec(ctx, q, *deviceID, *schoolID, hash); err != nil {
		log.Fatalf("insert device credential: %v", err)
	}

	fmt.Println("Device credential provisioned. This secret is shown once -- it is not recoverable from the database (only its bcrypt hash is stored).")
	fmt.Println()
	fmt.Printf("SCHOOL_BOX_ID=%s\n", *deviceID)
	fmt.Printf("SCHOOL_ID=%s\n", *schoolID)
	fmt.Printf("SCHOOL_BOX_SECRET=%s\n", secret)
	fmt.Println()
	fmt.Println("Paste these into that box's .env.school-box before its first boot.")
}

func generateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
