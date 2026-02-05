package httpapi

import (
	"context"
	"encoding/json"

	"pixia-panel/internal/outbox"
)

// EnqueueGost enqueues a Gost action into outbox.
func (s *Server) EnqueueGost(ctx context.Context, nodeID int64, action string, data json.RawMessage) error {
	payload := outbox.GostMessage{NodeID: nodeID, Action: action, Data: data}
	b, _ := json.Marshal(payload)
	_, err := s.store.EnqueueOutbox(ctx, action, b)
	return err
}
