package httpapi

import (
	"encoding/json"
	"net/http"

	"pixia-panel/internal/flow"
)

type FlowHandler struct {
	service *flow.Service
}

func NewFlowHandler(service *flow.Service) *FlowHandler {
	return &FlowHandler{service: service}
}

type flowRequest struct {
	ForwardID    int64 `json:"forward_id"`
	UserID       int64 `json:"user_id"`
	UserTunnelID int64 `json:"user_tunnel_id"`
	D            int64 `json:"d"`
	U            int64 `json:"u"`
}

func (h *FlowHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req flowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	update := flow.Update{
		ForwardID:    req.ForwardID,
		UserID:       req.UserID,
		UserTunnelID: req.UserTunnelID,
		Down:         req.D,
		Up:           req.U,
	}

	if err := h.service.Apply(r.Context(), update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
