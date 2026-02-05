package store

import (
	"context"
	"database/sql"
	"errors"
)

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status FROM user WHERE id = ?`, id)
	return scanUser(row)
}

func (s *Store) GetUserByName(ctx context.Context, username string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status FROM user WHERE user = ?`, username)
	return scanUser(row)
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status FROM user ORDER BY id`) 
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}
	return users, rows.Err()
}

func (s *Store) InsertUser(ctx context.Context, user *User) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO user(user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.User, user.Pwd, user.RoleID, user.ExpTime, user.Flow, user.InFlow, user.OutFlow, user.FlowResetTime, user.Num, user.CreatedTime, user.UpdatedTime, user.Status)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *Store) UpdateUser(ctx context.Context, user *User) error {
	_, err := s.db.ExecContext(ctx, `UPDATE user SET user = ?, pwd = ?, role_id = ?, exp_time = ?, flow = ?, in_flow = ?, out_flow = ?, flow_reset_time = ?, num = ?, updated_time = ?, status = ? WHERE id = ?`,
		user.User, user.Pwd, user.RoleID, user.ExpTime, user.Flow, user.InFlow, user.OutFlow, user.FlowResetTime, user.Num, user.UpdatedTime, user.Status, user.ID)
	return err
}

func (s *Store) UpdateUserFields(ctx context.Context, id int64, user string, pwd *string, flow int64, num int64, expTime int64, flowReset int64, status int64, updated int64) error {
	if pwd != nil {
		_, err := s.db.ExecContext(ctx, `UPDATE user SET user = ?, pwd = ?, flow = ?, num = ?, exp_time = ?, flow_reset_time = ?, status = ?, updated_time = ? WHERE id = ?`,
			user, *pwd, flow, num, expTime, flowReset, status, updated, id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE user SET user = ?, flow = ?, num = ?, exp_time = ?, flow_reset_time = ?, status = ?, updated_time = ? WHERE id = ?`,
		user, flow, num, expTime, flowReset, status, updated, id)
	return err
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user WHERE id = ?`, id)
	return err
}

func scanUser(scanner interface{ Scan(dest ...any) error }) (*User, error) {
	var user User
	var updated sql.NullInt64
	if err := scanner.Scan(&user.ID, &user.User, &user.Pwd, &user.RoleID, &user.ExpTime, &user.Flow, &user.InFlow, &user.OutFlow, &user.FlowResetTime, &user.Num, &user.CreatedTime, &updated, &user.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if updated.Valid {
		user.UpdatedTime = &updated.Int64
	}
	return &user, nil
}
