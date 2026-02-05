package store

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Store) GetSpeedLimitByID(ctx context.Context, id int64) (*SpeedLimit, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, speed, tunnel_id, tunnel_name, created_time, updated_time, status FROM speed_limit WHERE id = ?`, id)
	return scanSpeedLimit(row)
}

func (s *Store) ListSpeedLimits(ctx context.Context) ([]SpeedLimit, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, speed, tunnel_id, tunnel_name, created_time, updated_time, status FROM speed_limit ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SpeedLimit
	for rows.Next() {
		item, err := scanSpeedLimit(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *item)
	}
	return list, rows.Err()
}

func (s *Store) GetActiveSpeedLimitByTunnel(ctx context.Context, tunnelID int64) (*SpeedLimit, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, speed, tunnel_id, tunnel_name, created_time, updated_time, status FROM speed_limit WHERE tunnel_id = ? AND status = 1 ORDER BY id LIMIT 1`, tunnelID)
	return scanSpeedLimit(row)
}

func (s *Store) ListActiveSpeedLimitsByTunnel(ctx context.Context, tunnelID int64) ([]SpeedLimit, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, speed, tunnel_id, tunnel_name, created_time, updated_time, status FROM speed_limit WHERE tunnel_id = ? AND status = 1 ORDER BY id`, tunnelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SpeedLimit
	for rows.Next() {
		item, err := scanSpeedLimit(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *item)
	}
	return list, rows.Err()
}

func (s *Store) InsertSpeedLimit(ctx context.Context, limit *SpeedLimit) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO speed_limit(name, speed, tunnel_id, tunnel_name, created_time, updated_time, status)
		VALUES(?, ?, ?, ?, ?, ?, ?)`, limit.Name, limit.Speed, limit.TunnelID, limit.TunnelName, limit.CreatedTime, limit.UpdatedTime, limit.Status)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *Store) UpdateSpeedLimit(ctx context.Context, limit *SpeedLimit) error {
	_, err := s.db.ExecContext(ctx, `UPDATE speed_limit SET name = ?, speed = ?, tunnel_id = ?, tunnel_name = ?, updated_time = ?, status = ? WHERE id = ?`,
		limit.Name, limit.Speed, limit.TunnelID, limit.TunnelName, limit.UpdatedTime, limit.Status, limit.ID)
	return err
}

func (s *Store) DeleteSpeedLimit(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM speed_limit WHERE id = ?`, id)
	return err
}

func scanSpeedLimit(scanner interface{ Scan(dest ...any) error }) (*SpeedLimit, error) {
	var limit SpeedLimit
	var updated sql.NullInt64
	if err := scanner.Scan(&limit.ID, &limit.Name, &limit.Speed, &limit.TunnelID, &limit.TunnelName, &limit.CreatedTime, &updated, &limit.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if updated.Valid {
		limit.UpdatedTime = &updated.Int64
	}
	return &limit, nil
}
