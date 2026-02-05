package store

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Store) GetTunnelByID(ctx context.Context, id int64) (*Tunnel, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, traffic_ratio, in_node_id, in_ip, out_node_id, out_ip, type, protocol, flow, tcp_listen_addr, udp_listen_addr, interface_name, created_time, updated_time, status FROM tunnel WHERE id = ?`, id)
	return scanTunnel(row)
}

func (s *Store) GetTunnelByName(ctx context.Context, name string) (*Tunnel, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, traffic_ratio, in_node_id, in_ip, out_node_id, out_ip, type, protocol, flow, tcp_listen_addr, udp_listen_addr, interface_name, created_time, updated_time, status FROM tunnel WHERE name = ?`, name)
	return scanTunnel(row)
}

func (s *Store) ListTunnels(ctx context.Context) ([]Tunnel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, traffic_ratio, in_node_id, in_ip, out_node_id, out_ip, type, protocol, flow, tcp_listen_addr, udp_listen_addr, interface_name, created_time, updated_time, status FROM tunnel ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tunnels []Tunnel
	for rows.Next() {
		tunnel, err := scanTunnel(rows)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, *tunnel)
	}
	return tunnels, rows.Err()
}

func (s *Store) InsertTunnel(ctx context.Context, tunnel *Tunnel) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO tunnel(name, traffic_ratio, in_node_id, in_ip, out_node_id, out_ip, type, protocol, flow, tcp_listen_addr, udp_listen_addr, interface_name, created_time, updated_time, status)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tunnel.Name, tunnel.TrafficRatio, tunnel.InNodeID, tunnel.InIP, tunnel.OutNodeID, tunnel.OutIP, tunnel.Type, tunnel.Protocol, tunnel.Flow, tunnel.TCPListenAddr, tunnel.UDPListenAddr, tunnel.InterfaceName, tunnel.CreatedTime, tunnel.UpdatedTime, tunnel.Status)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *Store) UpdateTunnel(ctx context.Context, tunnel *Tunnel) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tunnel SET name = ?, traffic_ratio = ?, protocol = ?, flow = ?, tcp_listen_addr = ?, udp_listen_addr = ?, interface_name = ?, updated_time = ?, status = ? WHERE id = ?`,
		tunnel.Name, tunnel.TrafficRatio, tunnel.Protocol, tunnel.Flow, tunnel.TCPListenAddr, tunnel.UDPListenAddr, tunnel.InterfaceName, tunnel.UpdatedTime, tunnel.Status, tunnel.ID)
	return err
}

func (s *Store) DeleteTunnel(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tunnel WHERE id = ?`, id)
	return err
}

func (s *Store) CountTunnelsByInNode(ctx context.Context, nodeID int64) (int64, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM tunnel WHERE in_node_id = ?`, nodeID)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

func (s *Store) CountTunnelsByOutNode(ctx context.Context, nodeID int64) (int64, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM tunnel WHERE out_node_id = ?`, nodeID)
	var c int64
	if err := row.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

func (s *Store) UpdateTunnelsInIP(ctx context.Context, nodeID int64, ip string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tunnel SET in_ip = ? WHERE in_node_id = ?`, ip, nodeID)
	return err
}

func (s *Store) UpdateTunnelsOutIP(ctx context.Context, nodeID int64, ip string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tunnel SET out_ip = ? WHERE out_node_id = ?`, ip, nodeID)
	return err
}

func scanTunnel(scanner interface{ Scan(dest ...any) error }) (*Tunnel, error) {
	var tunnel Tunnel
	var iface sql.NullString
	if err := scanner.Scan(&tunnel.ID, &tunnel.Name, &tunnel.TrafficRatio, &tunnel.InNodeID, &tunnel.InIP, &tunnel.OutNodeID, &tunnel.OutIP, &tunnel.Type, &tunnel.Protocol, &tunnel.Flow, &tunnel.TCPListenAddr, &tunnel.UDPListenAddr, &iface, &tunnel.CreatedTime, &tunnel.UpdatedTime, &tunnel.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if iface.Valid {
		tunnel.InterfaceName = &iface.String
	}
	return &tunnel, nil
}
