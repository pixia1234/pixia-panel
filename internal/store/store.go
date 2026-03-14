package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	items, err := s.ClaimNextOutboxBatch(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	item := items[0]
	return &item, nil
}

func (s *Store) ClaimNextOutboxBatch(ctx context.Context, limit int) ([]OutboxItem, error) {
	if limit <= 0 {
		limit = 1
	}

	items := make([]OutboxItem, 0, limit)
	err := s.withImmediateTx(ctx, func(conn *sql.Conn) error {
		now := time.Now().UnixMilli()
		rows, err := conn.QueryContext(ctx, `SELECT id, type, payload, status, retry_count, next_retry_at, created_at, updated_at
			FROM outbox
			WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= ?)
			ORDER BY COALESCE(next_retry_at, 0), id
			LIMIT ?`, now, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		ids := make([]int64, 0, limit)
		for rows.Next() {
			var item OutboxItem
			var next sql.NullInt64
			if err := rows.Scan(&item.ID, &item.Type, &item.Payload, &item.Status, &item.RetryCount, &next, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return err
			}
			if next.Valid {
				item.NextRetryAt = &next.Int64
			}
			items = append(items, item)
			ids = append(ids, item.ID)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(ids) == 0 {
			return ErrNotFound
		}

		args := make([]any, 0, len(ids)+1)
		args = append(args, now)
		placeholders := make([]string, 0, len(ids))
		for _, id := range ids {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}

		query := "UPDATE outbox SET status = 'processing', updated_at = ? WHERE id IN (" + strings.Join(placeholders, ",") + ")"
		if _, err := conn.ExecContext(ctx, query, args...); err != nil {
			return err
		}

		for i := range items {
			items[i].Status = "processing"
			items[i].UpdatedAt = now
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Store) MarkOutboxSuccess(ctx context.Context, id int64) error {
	now := time.Now().UnixMilli()
	_, err := s.db.ExecContext(ctx, "UPDATE outbox SET status = 'done', next_retry_at = NULL, updated_at = ? WHERE id = ?", now, id)
	return err
}

func (s *Store) MarkOutboxFailed(ctx context.Context, id int64, delay time.Duration) error {
	now := time.Now().UnixMilli()
	next := now + delay.Milliseconds()
	_, err := s.db.ExecContext(ctx, "UPDATE outbox SET status = 'pending', retry_count = retry_count + 1, next_retry_at = ?, updated_at = ? WHERE id = ?", next, now, id)
	return err
}

func (s *Store) MarkOutboxDead(ctx context.Context, id int64, incrementRetry bool) error {
	now := time.Now().UnixMilli()
	query := "UPDATE outbox SET status = 'dead', next_retry_at = NULL, updated_at = ? WHERE id = ?"
	if incrementRetry {
		query = "UPDATE outbox SET status = 'dead', retry_count = retry_count + 1, next_retry_at = NULL, updated_at = ? WHERE id = ?"
	}
	_, err := s.db.ExecContext(ctx, query, now, id)
	return err
}

func (s *Store) RequeueStaleOutboxProcessing(ctx context.Context, maxAge time.Duration) (int64, error) {
	if maxAge <= 0 {
		return 0, nil
	}

	now := time.Now().UnixMilli()
	cutoff := now - maxAge.Milliseconds()
	res, err := s.db.ExecContext(ctx, "UPDATE outbox SET status = 'pending', updated_at = ? WHERE status = 'processing' AND updated_at <= ?", now, cutoff)
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	return affected, nil
}

func (s *Store) MarkOutboxDeadByNodeID(ctx context.Context, nodeID int64) (int64, error) {
	if nodeID <= 0 {
		return 0, nil
	}

	type outboxMessage struct {
		NodeID int64 `json:"node_id"`
	}

	ids := make([]int64, 0, 16)
	err := s.withImmediateTx(ctx, func(conn *sql.Conn) error {
		rows, err := conn.QueryContext(ctx, `SELECT id, payload FROM outbox WHERE status IN ('pending', 'processing')`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			var payload []byte
			if err := rows.Scan(&id, &payload); err != nil {
				return err
			}

			var msg outboxMessage
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}
			if msg.NodeID == nodeID {
				ids = append(ids, id)
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}

		now := time.Now().UnixMilli()
		args := make([]any, 0, len(ids)+1)
		args = append(args, now)
		placeholders := make([]string, 0, len(ids))
		for _, id := range ids {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}

		query := "UPDATE outbox SET status = 'dead', next_retry_at = NULL, updated_at = ? WHERE id IN (" + strings.Join(placeholders, ",") + ")"
		if _, err := conn.ExecContext(ctx, query, args...); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return int64(len(ids)), nil
}
