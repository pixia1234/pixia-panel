package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *Store) GetConfigByName(ctx context.Context, name string) (*ViteConfig, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, value, time FROM vite_config WHERE name = ?`, name)
	var cfg ViteConfig
	if err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Value, &cfg.Time); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &cfg, nil
}

func (s *Store) ListConfigs(ctx context.Context) ([]ViteConfig, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, value, time FROM vite_config ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ViteConfig
	for rows.Next() {
		var cfg ViteConfig
		if err := rows.Scan(&cfg.ID, &cfg.Name, &cfg.Value, &cfg.Time); err != nil {
			return nil, err
		}
		list = append(list, cfg)
	}
	return list, rows.Err()
}

func (s *Store) UpsertConfig(ctx context.Context, name, value string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.ExecContext(ctx, `INSERT INTO vite_config(name, value, time) VALUES(?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET value = excluded.value, time = excluded.time`, name, value, now)
	return err
}
