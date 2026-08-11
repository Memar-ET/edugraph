// db/scripts/seed-users.go
// Standalone script to seed the database with demo users.
// Run from backend folder: go run db/scripts/seed-users.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type DemoUser struct {
	Email    string
	Password string
	Role     string
	FullName string
	Phone    string
}

func main() {
	// 1. Load .env file from project root (one level up from backend)
	envPath := filepath.Join("..", ".env")
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("⚠️  Warning: Could not load .env file from %s: %v\n", envPath, err)
		log.Println("Falling back to environment variables...")
	}

	// 2. Construct DATABASE_URL from .env variables
	pgHost := getEnv("POSTGRES_HOST", "localhost")
	pgPort := getEnv("POSTGRES_PORT", "5432")
	pgUser := getEnv("POSTGRES_USER", "postgres")
	pgPass := getEnv("POSTGRES_PASSWORD", "postgres")
	pgDB := getEnv("POSTGRES_DB", "edugraph")

	// Smart host detection: if running outside Docker, use localhost
	// If POSTGRES_HOST is "postgres" (Docker service name), try localhost first
	if pgHost == "postgres" || pgHost == "db" {
		log.Println("ℹ️  Detected Docker service name in POSTGRES_HOST. Using 'localhost' for local execution.")
		pgHost = "localhost"
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pgUser, pgPass, pgHost, pgPort, pgDB)

	fmt.Printf("🔗 Connecting to: postgres://%s:%s@%s:%s/%s\n",
		pgUser, "****", pgHost, pgPort, pgDB)

	// 3. Connect to PostgreSQL
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	fmt.Println("✅ Connected to database. Generating demo users...")

	// 4. Define the users we want to create
	users := []DemoUser{
		{Email: "ministry.admin@edugraph.et", Password: "password123", Role: "ministry_admin", FullName: "Dr. Abebe Kebede", Phone: "+251911234567"},
		{Email: "curriculum.officer@edugraph.et", Password: "password123", Role: "curriculum_officer", FullName: "Selamawit Tesfaye", Phone: "+251911234568"},
		{Email: "regional.admin@edugraph.et", Password: "password123", Role: "regional_admin", FullName: "Mohammed Ahmed", Phone: "+251911234569"},
		{Email: "school.admin@edugraph.et", Password: "password123", Role: "school_admin", FullName: "Tigist Bekele", Phone: "+251911234570"},
		{Email: "teacher@edugraph.et", Password: "password123", Role: "teacher", FullName: "Dawit Lemma", Phone: "+251911234571"},
		{Email: "student@edugraph.et", Password: "password123", Role: "student", FullName: "Hanna Solomon", Phone: "+251911234572"},
	}

	// 5. Loop through users, hash password, and insert
	successCount := 0
	for _, u := range users {
		// Hash the password using bcrypt (Cost 12 is the industry standard)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), 12)
		if err != nil {
			log.Printf("❌ Error hashing password for %s: %v\n", u.Email, err)
			continue
		}

		// Insert into the database (using ON CONFLICT so we can run this script multiple times safely)
		sql := `
			INSERT INTO users (email, password_hash, role, full_name, phone, is_active)
			VALUES ($1, $2, $3, $4, $5, true)
			ON CONFLICT (email) DO NOTHING;
		`

		_, err = pool.Exec(context.Background(), sql, u.Email, string(hashedPassword), u.Role, u.FullName, u.Phone)
		if err != nil {
			log.Printf("❌ Failed to insert %s: %v\n", u.Email, err)
		} else {
			fmt.Printf("✅ Created user: %s (%s)\n", u.Email, u.Role)
			successCount++
		}
	}

	fmt.Printf("\n🎉 Demo users seeded successfully! (%d/%d created)\n", successCount, len(users))
	fmt.Println("🔑 Password for all users is: password123")
	fmt.Println("===========================================")
}

// getEnv reads an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
