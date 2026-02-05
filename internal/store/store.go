package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

// withImmediateTx runs fn within a BEGIN IMMEDIATE transaction using a dedicated connection.
func (s *Store) withImmediateTx(ctx context.Context, fn func(*sql.Conn) error) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}

	err = fn(conn)
	if err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return err
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}

	return nil
}

// ApplyFlow atomically updates forward/user/user_tunnel flow stats.
func (s *Store) ApplyFlow(ctx context.Context, forwardID, userID, userTunnelID, d, u int64) error {
	return s.withImmediateTx(ctx, func(conn *sql.Conn) error {
		res, err := conn.ExecContext(ctx, "UPDATE forward SET in_flow = in_flow + ?, out_flow = out_flow + ? WHERE id = ?", d, u, forwardID)
		if err != nil {
			return err
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return fmt.Errorf("forward %d: %w", forwardID, ErrNotFound)
		}

		res, err = conn.ExecContext(ctx, "UPDATE user SET in_flow = in_flow + ?, out_flow = out_flow + ? WHERE id = ?", d, u, userID)
		if err != nil {
			return err
		}
		affected, _ = res.RowsAffected()
		if affected == 0 {
			return fmt.Errorf("user %d: %w", userID, ErrNotFound)
		}

		if userTunnelID != 0 {
			res, err = conn.ExecContext(ctx, "UPDATE user_tunnel SET in_flow = in_flow + ?, out_flow = out_flow + ? WHERE id = ?", d, u, userTunnelID)
			if err != nil {
				return err
			}
			affected, _ = res.RowsAffected()
			if affected == 0 {
				return fmt.Errorf("user_tunnel %d: %w", userTunnelID, ErrNotFound)
			}
		}

		return nil
	})
}

// Outbox
func (s *Store) EnqueueOutbox(ctx context.Context, typ string, payload json.RawMessage) (int64, error) {
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, "INSERT INTO outbox(type, payload, status, retry_count, next_retry_at, created_at, updated_at) VALUES(?, ?, 'pending', 0, NULL, ?, ?)", typ, payload, now, now)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *Store) ClaimNextOutbox(ctx context.Context) (*OutboxItem, error) {
	var item *OutboxItem
	err := s.withImmediateTx(ctx, func(conn *sql.Conn) error {
		now := time.Now().UnixMilli()
		row := conn.QueryRowContext(ctx, `SELECT id, type, payload, status, retry_count, next_retry_at, created_at, updated_at
			FROM outbox
			WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= ?)
			ORDER BY id LIMIT 1`, now)

		var tmp OutboxItem
		var next sql.NullInt64
		if err := row.Scan(&tmp.ID, &tmp.Type, &tmp.Payload, &tmp.Status, &tmp.RetryCount, &next, &tmp.CreatedAt, &tmp.UpdatedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if next.Valid {
			tmp.NextRetryAt = &next.Int64
		}

		if _, err := conn.ExecContext(ctx, "UPDATE outbox SET status = 'processing', updated_at = ? WHERE id = ?", now, tmp.ID); err != nil {
			return err
		}

		item = &tmp
		return nil
	})

	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Store) MarkOutboxSuccess(ctx context.Context, id int64) error {
	now := time.Now().UnixMilli()
	_, err := s.db.ExecContext(ctx, "UPDATE outbox SET status = 'done', updated_at = ? WHERE id = ?", now, id)
	return err
}

func (s *Store) MarkOutboxFailed(ctx context.Context, id int64, delay time.Duration) error {
	now := time.Now().UnixMilli()
	next := now + delay.Milliseconds()
	_, err := s.db.ExecContext(ctx, "UPDATE outbox SET status = 'pending', retry_count = retry_count + 1, next_retry_at = ?, updated_at = ? WHERE id = ?", next, now, id)
	return err
}
