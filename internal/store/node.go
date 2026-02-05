package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *Store) GetNodeByID(ctx context.Context, id int64) (*Node, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, secret, ip, server_ip, port_sta, port_end, version, http, tls, socks, created_time, updated_time, status FROM node WHERE id = ?`, id)
	return scanNode(row)
}

func (s *Store) GetNodeBySecret(ctx context.Context, secret string) (*Node, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, secret, ip, server_ip, port_sta, port_end, version, http, tls, socks, created_time, updated_time, status FROM node WHERE secret = ?`, secret)
	return scanNode(row)
}

func (s *Store) ListNodes(ctx context.Context) ([]Node, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, secret, ip, server_ip, port_sta, port_end, version, http, tls, socks, created_time, updated_time, status FROM node ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}
	return nodes, rows.Err()
}

func (s *Store) InsertNode(ctx context.Context, node *Node) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO node(name, secret, ip, server_ip, port_sta, port_end, version, http, tls, socks, created_time, updated_time, status)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		node.Name, node.Secret, node.IP, node.ServerIP, node.PortSta, node.PortEnd, node.Version, node.HTTP, node.TLS, node.Socks, node.CreatedTime, node.UpdatedTime, node.Status)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *Store) UpdateNode(ctx context.Context, node *Node) error {
	_, err := s.db.ExecContext(ctx, `UPDATE node SET name = ?, ip = ?, server_ip = ?, port_sta = ?, port_end = ?, version = ?, http = ?, tls = ?, socks = ?, updated_time = ?, status = ? WHERE id = ?`,
		node.Name, node.IP, node.ServerIP, node.PortSta, node.PortEnd, node.Version, node.HTTP, node.TLS, node.Socks, node.UpdatedTime, node.Status, node.ID)
	return err
}

func (s *Store) UpdateNodeStatus(ctx context.Context, id int64, status int64, version *string, httpVal, tlsVal, socksVal *int64) error {
	now := time.Now().UnixMilli()
	var v any = nil
	if version != nil {
		v = *version
	}
	var httpAny any = nil
	if httpVal != nil {
		httpAny = *httpVal
	}
	var tlsAny any = nil
	if tlsVal != nil {
		tlsAny = *tlsVal
	}
	var socksAny any = nil
	if socksVal != nil {
		socksAny = *socksVal
	}
	_, err := s.db.ExecContext(ctx, `UPDATE node SET status = ?, version = COALESCE(?, version), http = COALESCE(?, http), tls = COALESCE(?, tls), socks = COALESCE(?, socks), updated_time = ? WHERE id = ?`,
		status, v, httpAny, tlsAny, socksAny, now, id)
	return err
}

func (s *Store) DeleteNode(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM node WHERE id = ?`, id)
	return err
}

func scanNode(scanner interface{ Scan(dest ...any) error }) (*Node, error) {
	var node Node
	var ip sql.NullString
	var version sql.NullString
	var updated sql.NullInt64
	if err := scanner.Scan(&node.ID, &node.Name, &node.Secret, &ip, &node.ServerIP, &node.PortSta, &node.PortEnd, &version, &node.HTTP, &node.TLS, &node.Socks, &node.CreatedTime, &updated, &node.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if ip.Valid {
		node.IP = &ip.String
	}
	if version.Valid {
		node.Version = &version.String
	}
	if updated.Valid {
		node.UpdatedTime = &updated.Int64
	}
	return &node, nil
}
