package flow

import (
	"context"
	"errors"

	"pixia-panel/internal/store"
)

type Service struct {
	store *store.Store
}

func New(store *store.Store) *Service {
	return &Service{store: store}
}

type Update struct {
	ForwardID    int64
	UserID       int64
	UserTunnelID int64
	Down         int64
	Up           int64
}

func (s *Service) Apply(ctx context.Context, update Update) error {
	if update.ForwardID == 0 || update.UserID == 0 {
		return errors.New("invalid flow update: missing IDs")
	}
	return s.store.ApplyFlow(ctx, update.ForwardID, update.UserID, update.UserTunnelID, update.Down, update.Up)
}
