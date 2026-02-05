package store

import (
	"context"
	"database/sql"
	"errors"
)

type ForwardWithTunnel struct {
	Forward
	TunnelName string `json:"tunnelName"`
	TunnelType int64  `json:"tunnelType"`
	InNodeID   int64  `json:"inNodeId"`
	OutNodeID  int64  `json:"outNodeId"`
}

func (s *Store) GetForwardByID(ctx context.Context, id int64) (*Forward, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, user_name, name, tunnel_id, in_port, out_port, remote_addr, strategy, interface_name, in_flow, out_flow, created_time, updated_time, status, inx, lifecycle FROM forward WHERE id = ?`, id)
	return scanForward(row)
}

func (s *Store) ListForwardsByUser(ctx context.Context, userID int64) ([]ForwardWithTunnel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT f.id, f.user_id, f.user_name, f.name, f.tunnel_id, f.in_port, f.out_port, f.remote_addr, f.strategy, f.interface_name, f.in_flow, f.out_flow, f.created_time, f.updated_time, f.status, f.inx, f.lifecycle,
		t.name, t.type, t.in_node_id, t.out_node_id
		FROM forward f JOIN tunnel t ON f.tunnel_id = t.id WHERE f.user_id = ? ORDER BY f.inx, f.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanForwardWithTunnelRows(rows)
}

func (s *Store) ListForwardsAll(ctx context.Context) ([]ForwardWithTunnel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT f.id, f.user_id, f.user_name, f.name, f.tunnel_id, f.in_port, f.out_port, f.remote_addr, f.strategy, f.interface_name, f.in_flow, f.out_flow, f.created_time, f.updated_time, f.status, f.inx, f.lifecycle,
		t.name, t.type, t.in_node_id, t.out_node_id
		FROM forward f JOIN tunnel t ON f.tunnel_id = t.id ORDER BY f.inx, f.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanForwardWithTunnelRows(rows)
}

func (s *Store) ListForwardsByTunnel(ctx context.Context, tunnelID int64) ([]Forward, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, user_name, name, tunnel_id, in_port, out_port, remote_addr, strategy, interface_name, in_flow, out_flow, created_time, updated_time, status, inx, lifecycle FROM forward WHERE tunnel_id = ?`, tunnelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Forward
	for rows.Next() {
		fw, err := scanForward(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *fw)
	}
	return list, rows.Err()
}

func (s *Store) InsertForward(ctx context.Context, forward *Forward) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO forward(user_id, user_name, name, tunnel_id, in_port, out_port, remote_addr, strategy, interface_name, in_flow, out_flow, created_time, updated_time, status, inx, lifecycle)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		forward.UserID, forward.UserName, forward.Name, forward.TunnelID, forward.InPort, forward.OutPort, forward.RemoteAddr, forward.Strategy, forward.InterfaceName, forward.InFlow, forward.OutFlow, forward.CreatedTime, forward.UpdatedTime, forward.Status, forward.Inx, forward.Lifecycle)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *Store) UpdateForward(ctx context.Context, forward *Forward) error {
	_, err := s.db.ExecContext(ctx, `UPDATE forward SET user_id = ?, user_name = ?, name = ?, tunnel_id = ?, in_port = ?, out_port = ?, remote_addr = ?, strategy = ?, interface_name = ?, updated_time = ?, status = ?, inx = ?, lifecycle = ? WHERE id = ?`,
		forward.UserID, forward.UserName, forward.Name, forward.TunnelID, forward.InPort, forward.OutPort, forward.RemoteAddr, forward.Strategy, forward.InterfaceName, forward.UpdatedTime, forward.Status, forward.Inx, forward.Lifecycle, forward.ID)
	return err
}

func (s *Store) UpdateForwardStatus(ctx context.Context, id int64, status int64, lifecycle string, updated int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE forward SET status = ?, lifecycle = ?, updated_time = ? WHERE id = ?`, status, lifecycle, updated, id)
	return err
}

func (s *Store) UpdateForwardOrder(ctx context.Context, id int64, inx int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE forward SET inx = ? WHERE id = ?`, inx, id)
	return err
}

func (s *Store) DeleteForward(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM forward WHERE id = ?`, id)
	return err
}

func (s *Store) CountForwardsByUser(ctx context.Context, userID int64) (int64, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM forward WHERE user_id = ?`, userID)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

func (s *Store) CountForwardsByUserTunnel(ctx context.Context, userID, tunnelID int64) (int64, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM forward WHERE user_id = ? AND tunnel_id = ?`, userID, tunnelID)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

func (s *Store) CountForwardsByTunnel(ctx context.Context, tunnelID int64) (int64, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM forward WHERE tunnel_id = ?`, tunnelID)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

func scanForward(scanner interface{ Scan(dest ...any) error }) (*Forward, error) {
	var forward Forward
	var outPort sql.NullInt64
	var iface sql.NullString
	if err := scanner.Scan(&forward.ID, &forward.UserID, &forward.UserName, &forward.Name, &forward.TunnelID, &forward.InPort, &outPort, &forward.RemoteAddr, &forward.Strategy, &iface, &forward.InFlow, &forward.OutFlow, &forward.CreatedTime, &forward.UpdatedTime, &forward.Status, &forward.Inx, &forward.Lifecycle); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if outPort.Valid {
		forward.OutPort = &outPort.Int64
	}
	if iface.Valid {
		forward.InterfaceName = &iface.String
	}
	return &forward, nil
}

func scanForwardWithTunnelRows(rows *sql.Rows) ([]ForwardWithTunnel, error) {
	var list []ForwardWithTunnel
	for rows.Next() {
		var fw ForwardWithTunnel
		var outPort sql.NullInt64
		var iface sql.NullString
		if err := rows.Scan(&fw.ID, &fw.UserID, &fw.UserName, &fw.Name, &fw.TunnelID, &fw.InPort, &outPort, &fw.RemoteAddr, &fw.Strategy, &iface, &fw.InFlow, &fw.OutFlow, &fw.CreatedTime, &fw.UpdatedTime, &fw.Status, &fw.Inx, &fw.Lifecycle,
			&fw.TunnelName, &fw.TunnelType, &fw.InNodeID, &fw.OutNodeID); err != nil {
			return nil, err
		}
		if outPort.Valid {
			fw.OutPort = &outPort.Int64
		}
		if iface.Valid {
			fw.InterfaceName = &iface.String
		}
		list = append(list, fw)
	}
	return list, rows.Err()
}
