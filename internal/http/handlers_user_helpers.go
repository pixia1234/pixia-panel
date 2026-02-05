package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"pixia-panel/internal/gost"
	"pixia-panel/internal/outbox"
)

func (s *Server) deleteUserCascade(r *http.Request, userID int64) error {
	forwards, err := s.store.ListForwardsByUser(r.Context(), userID)
	if err != nil {
		return err
	}

	for _, fw := range forwards {
		userTunnelID := s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID)
		name := buildServiceName(fw.ID, fw.UserID, userTunnelID)

		_ = s.enqueueGost(r, fw.InNodeID, "DeleteService", gost.DeleteServiceData(name))
		if fw.TunnelType == 2 {
			_ = s.enqueueGost(r, fw.InNodeID, "DeleteChains", gost.DeleteChainsData(name))
			_ = s.enqueueGost(r, fw.OutNodeID, "DeleteService", gost.DeleteRemoteServiceData(name))
		}
		_ = s.store.DeleteForward(r.Context(), fw.ID)
	}

	_ = s.store.DeleteStatisticsByUser(r.Context(), userID)
	if err := s.store.DeleteUser(r.Context(), userID); err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	// user_tunnel rows will be cascade deleted by FK.
	return nil
}

func (s *Server) enqueueGost(r *http.Request, nodeID int64, action string, data json.RawMessage) error {
	payload := outbox.GostMessage{NodeID: nodeID, Action: action, Data: data}
	b, _ := json.Marshal(payload)
	_, err := s.store.EnqueueOutbox(r.Context(), action, b)
	return err
}
