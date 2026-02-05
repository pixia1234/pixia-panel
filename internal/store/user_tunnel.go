package store

import (
	"context"
	"database/sql"
	"errors"
)

type UserTunnelDetail struct {
	UserTunnel
	TunnelName string `json:"tunnelName"`
	TunnelType int64  `json:"tunnelType"`
}

func (s *Store) GetUserTunnelByID(ctx context.Context, id int64) (*UserTunnel, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status FROM user_tunnel WHERE id = ?`, id)
	return scanUserTunnel(row)
}

func (s *Store) GetUserTunnelByUserAndTunnel(ctx context.Context, userID, tunnelID int64) (*UserTunnel, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status FROM user_tunnel WHERE user_id = ? AND tunnel_id = ?`, userID, tunnelID)
	return scanUserTunnel(row)
}

func (s *Store) ListUserTunnelsByUser(ctx context.Context, userID int64) ([]UserTunnelDetail, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ut.id, ut.user_id, ut.tunnel_id, ut.speed_id, ut.num, ut.flow, ut.in_flow, ut.out_flow, ut.flow_reset_time, ut.exp_time, ut.status, t.name, t.type
		FROM user_tunnel ut JOIN tunnel t ON ut.tunnel_id = t.id WHERE ut.user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []UserTunnelDetail
	for rows.Next() {
		var item UserTunnelDetail
		var speed sql.NullInt64
		if err := rows.Scan(&item.ID, &item.UserID, &item.TunnelID, &speed, &item.Num, &item.Flow, &item.InFlow, &item.OutFlow, &item.FlowResetTime, &item.ExpTime, &item.Status, &item.TunnelName, &item.TunnelType); err != nil {
			return nil, err
		}
		if speed.Valid {
			item.SpeedID = &speed.Int64
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (s *Store) InsertUserTunnel(ctx context.Context, ut *UserTunnel) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO user_tunnel(user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ut.UserID, ut.TunnelID, ut.SpeedID, ut.Num, ut.Flow, ut.InFlow, ut.OutFlow, ut.FlowResetTime, ut.ExpTime, ut.Status)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *Store) UpdateUserTunnel(ctx context.Context, ut *UserTunnel) error {
	_, err := s.db.ExecContext(ctx, `UPDATE user_tunnel SET speed_id = ?, num = ?, flow = ?, in_flow = ?, out_flow = ?, flow_reset_time = ?, exp_time = ?, status = ? WHERE id = ?`,
		ut.SpeedID, ut.Num, ut.Flow, ut.InFlow, ut.OutFlow, ut.FlowResetTime, ut.ExpTime, ut.Status, ut.ID)
	return err
}

func (s *Store) DeleteUserTunnel(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_tunnel WHERE id = ?`, id)
	return err
}

func (s *Store) CountUserTunnelsByTunnel(ctx context.Context, tunnelID int64) (int64, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM user_tunnel WHERE tunnel_id = ?`, tunnelID)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

func (s *Store) CountUserTunnelsBySpeed(ctx context.Context, speedID int64) (int64, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM user_tunnel WHERE speed_id = ?`, speedID)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

func scanUserTunnel(scanner interface{ Scan(dest ...any) error }) (*UserTunnel, error) {
	var ut UserTunnel
	var speed sql.NullInt64
	if err := scanner.Scan(&ut.ID, &ut.UserID, &ut.TunnelID, &speed, &ut.Num, &ut.Flow, &ut.InFlow, &ut.OutFlow, &ut.FlowResetTime, &ut.ExpTime, &ut.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if speed.Valid {
		ut.SpeedID = &speed.Int64
	}
	return &ut, nil
}
