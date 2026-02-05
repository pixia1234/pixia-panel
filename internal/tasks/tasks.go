package tasks

import (
	"context"
	"strconv"
	"time"

	"pixia-panel/internal/gost"
	httpapi "pixia-panel/internal/http"
	"pixia-panel/internal/store"
)

type Scheduler struct {
	store *store.Store
	api   *httpapi.Server
}

func New(store *store.Store, api *httpapi.Server) *Scheduler {
	return &Scheduler{store: store, api: api}
}

// HourlyStatistics collects per-user flow stats and trims old records.
func (s *Scheduler) HourlyStatistics(ctx context.Context) {
	now := time.Now()
	cutoff := now.Add(-48 * time.Hour).UnixMilli()
	_ = s.store.DeleteStatisticsOlderThan(ctx, cutoff)

	rows, err := s.store.DB().QueryContext(ctx, `SELECT id, in_flow, out_flow FROM user`)
	if err != nil {
		return
	}
	defer rows.Close()

	var items []store.StatisticsFlow
	hourString := now.Format("15:04")
	created := now.UnixMilli()
	for rows.Next() {
		var userID, inFlow, outFlow int64
		if err := rows.Scan(&userID, &inFlow, &outFlow); err != nil {
			continue
		}
		total := inFlow + outFlow

		var lastTotal int64
		row := s.store.DB().QueryRowContext(ctx, `SELECT total_flow FROM statistics_flow WHERE user_id = ? ORDER BY id DESC LIMIT 1`, userID)
		_ = row.Scan(&lastTotal)
		increment := total - lastTotal
		if increment < 0 {
			increment = total
		}

		items = append(items, store.StatisticsFlow{
			UserID:      userID,
			Flow:        increment,
			TotalFlow:   total,
			Time:        hourString,
			CreatedTime: created,
		})
	}
	_ = s.store.InsertStatistics(ctx, items)
}

// DailyReset resets flows and handles expiration.
func (s *Scheduler) DailyReset(ctx context.Context) {
	today := time.Now()
	day := today.Day()
	lastDay := daysInMonth(today)

	_, _ = s.store.DB().ExecContext(ctx, `UPDATE user SET in_flow = 0, out_flow = 0 WHERE flow_reset_time != 0 AND (flow_reset_time = ? OR (flow_reset_time > ? AND ? = ?))`, day, lastDay, day, lastDay)
	_, _ = s.store.DB().ExecContext(ctx, `UPDATE user_tunnel SET in_flow = 0, out_flow = 0 WHERE flow_reset_time != 0 AND (flow_reset_time = ? OR (flow_reset_time > ? AND ? = ?))`, day, lastDay, day, lastDay)

	s.expireUsers(ctx)
	s.expireUserTunnels(ctx)
}

func (s *Scheduler) expireUsers(ctx context.Context) {
	now := time.Now().UnixMilli()
	rows, err := s.store.DB().QueryContext(ctx, `SELECT id FROM user WHERE role_id != 0 AND status = 1 AND exp_time IS NOT NULL AND exp_time < ?`, now)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		_, _ = s.store.DB().ExecContext(ctx, `UPDATE user SET status = 0 WHERE id = ?`, id)
		s.pauseUserForwards(ctx, id)
	}
}

func (s *Scheduler) expireUserTunnels(ctx context.Context) {
	now := time.Now().UnixMilli()
	rows, err := s.store.DB().QueryContext(ctx, `SELECT id, user_id, tunnel_id FROM user_tunnel WHERE status = 1 AND exp_time IS NOT NULL AND exp_time < ?`, now)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, userID, tunnelID int64
		if err := rows.Scan(&id, &userID, &tunnelID); err != nil {
			continue
		}
		_, _ = s.store.DB().ExecContext(ctx, `UPDATE user_tunnel SET status = 0 WHERE id = ?`, id)
		s.pauseUserTunnelForwards(ctx, userID, tunnelID)
	}
}

func (s *Scheduler) pauseUserForwards(ctx context.Context, userID int64) {
	forwards, err := s.store.ListForwardsByUser(ctx, userID)
	if err != nil {
		return
	}
	for _, fw := range forwards {
		name := buildServiceName(fw.ID, fw.UserID, s.resolveUserTunnelID(ctx, fw.UserID, fw.TunnelID))
		_ = s.api.EnqueueGost(ctx, fw.InNodeID, "PauseService", gost.PauseServiceData(name))
		if fw.TunnelType == 2 {
			_ = s.api.EnqueueGost(ctx, fw.OutNodeID, "PauseService", gost.PauseRemoteServiceData(name))
		}
		_ = s.store.UpdateForwardStatus(ctx, fw.ID, 0, "paused", time.Now().UnixMilli())
	}
}

func (s *Scheduler) pauseUserTunnelForwards(ctx context.Context, userID, tunnelID int64) {
	forwards, err := s.store.ListForwardsByUser(ctx, userID)
	if err != nil {
		return
	}
	for _, fw := range forwards {
		if fw.TunnelID != tunnelID {
			continue
		}
		name := buildServiceName(fw.ID, fw.UserID, s.resolveUserTunnelID(ctx, fw.UserID, fw.TunnelID))
		_ = s.api.EnqueueGost(ctx, fw.InNodeID, "PauseService", gost.PauseServiceData(name))
		if fw.TunnelType == 2 {
			_ = s.api.EnqueueGost(ctx, fw.OutNodeID, "PauseService", gost.PauseRemoteServiceData(name))
		}
		_ = s.store.UpdateForwardStatus(ctx, fw.ID, 0, "paused", time.Now().UnixMilli())
	}
}

func daysInMonth(t time.Time) int {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	return first.AddDate(0, 1, -1).Day()
}

func buildServiceName(forwardID, userID, userTunnelID int64) string {
	return strconv.FormatInt(forwardID, 10) + "_" + strconv.FormatInt(userID, 10) + "_" + strconv.FormatInt(userTunnelID, 10)
}

func (s *Scheduler) resolveUserTunnelID(ctx context.Context, userID, tunnelID int64) int64 {
	ut, err := s.store.GetUserTunnelByUserAndTunnel(ctx, userID, tunnelID)
	if err != nil {
		return 0
	}
	return ut.ID
}
