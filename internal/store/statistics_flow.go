package store

import (
	"context"
	"database/sql"
)

func (s *Store) ListRecentStatistics(ctx context.Context, userID int64, limit int) ([]StatisticsFlow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, flow, total_flow, time, created_time FROM statistics_flow WHERE user_id = ? ORDER BY id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []StatisticsFlow
	for rows.Next() {
		var sf StatisticsFlow
		if err := rows.Scan(&sf.ID, &sf.UserID, &sf.Flow, &sf.TotalFlow, &sf.Time, &sf.CreatedTime); err != nil {
			return nil, err
		}
		list = append(list, sf)
	}
	return list, rows.Err()
}

func (s *Store) InsertStatistics(ctx context.Context, items []StatisticsFlow) error {
	if len(items) == 0 {
		return nil
	}
	return s.withImmediateTx(ctx, func(conn *sql.Conn) error {
		stmt, err := conn.PrepareContext(ctx, `INSERT INTO statistics_flow(user_id, flow, total_flow, time, created_time) VALUES(?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, item := range items {
			if _, err := stmt.ExecContext(ctx, item.UserID, item.Flow, item.TotalFlow, item.Time, item.CreatedTime); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) DeleteStatisticsOlderThan(ctx context.Context, cutoff int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM statistics_flow WHERE created_time < ?`, cutoff)
	return err
}

func (s *Store) DeleteStatisticsByUser(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM statistics_flow WHERE user_id = ?`, userID)
	return err
}
