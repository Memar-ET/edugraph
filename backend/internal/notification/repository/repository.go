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

type Notification struct {
	ID        string
	UserID    string
	Title     string
	Body      string
	IsRead    bool
	CreatedAt time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID, title, body string) (Notification, error) {
	const q = `INSERT INTO notifications (user_id, title, body) VALUES ($1, $2, $3)
		RETURNING id, user_id, title, body, is_read, created_at`
	return scan(r.pool.QueryRow(ctx, q, userID, title, body))
}

func (r *Repository) ListForUser(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int64, error) {
	q := `SELECT id, user_id, title, body, is_read, created_at FROM notifications WHERE user_id = $1`
	countQ := `SELECT count(*) FROM notifications WHERE user_id = $1`
	if unreadOnly {
		q += ` AND NOT is_read`
		countQ += ` AND NOT is_read`
	}
	q += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, limit, offset)

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var items []Notification
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, n)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	return items, total, nil
}

func (r *Repository) MarkRead(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	if err := row.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.IsRead, &n.CreatedAt); err != nil {
		return Notification{}, fmt.Errorf("scan notification: %w", err)
	}
	return n, nil
}
