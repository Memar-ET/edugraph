package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	FullName     string
	Phone        *string
	RegionID     *string
	SchoolID     *string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateUserParams struct {
	Email        string
	PasswordHash string
	Role         string
	FullName     string
	Phone        *string
	RegionID     *string
	SchoolID     *string
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, p CreateUserParams) (User, error) {
	const q = `
		INSERT INTO users (email, password_hash, role, full_name, phone, region_id, school_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, password_hash, role, full_name, phone, region_id, school_id, is_active, created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, p.Email, p.PasswordHash, p.Role, p.FullName, p.Phone, p.RegionID, p.SchoolID)
	return scanUser(row)
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	const q = `
		SELECT id, email, password_hash, role, full_name, phone, region_id, school_id, is_active, created_at, updated_at
		FROM users WHERE email = $1`
	row := r.pool.QueryRow(ctx, q, email)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (User, error) {
	const q = `
		SELECT id, email, password_hash, role, full_name, phone, region_id, school_id, is_active, created_at, updated_at
		FROM users WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &u.Phone,
		&u.RegionID, &u.SchoolID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}

// ── Refresh tokens ────────────────────────────────────────────

func (r *Repository) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	const q = `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, q, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (r *Repository) GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error) {
	const q = `SELECT id, user_id, token_hash, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = $1`
	var t RefreshToken
	err := r.pool.QueryRow(ctx, q, tokenHash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshToken{}, ErrNotFound
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("get refresh token: %w", err)
	}
	return t, nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	const q = `UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1`
	if _, err := r.pool.Exec(ctx, q, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}
