package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pixia-panel/internal/crypto"
	"pixia-panel/internal/flow"
	"pixia-panel/internal/gost"
)

type flowDTO struct {
	N string `json:"n"`
	U int64  `json:"u"`
	D int64  `json:"d"`
}

type encryptedMessage struct {
	Encrypted bool   `json:"encrypted"`
	Data      string `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

type configItem struct {
	Name string `json:"name"`
}

type gostConfig struct {
	Services []configItem `json:"services"`
	Chains   []configItem `json:"chains"`
	Limiters []configItem `json:"limiters"`
}

func (s *Server) handleFlowTest(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("test"))
}

func (s *Server) handleFlowConfig(w http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("secret")
	if secret == "" {
		writeJSON(w, http.StatusUnauthorized, Err("缺少secret"))
		return
	}
	node, err := s.store.GetNodeBySecret(r.Context(), secret)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Err("节点不存在"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("读取失败"))
		return
	}
	if len(body) == 0 {
		writeJSON(w, http.StatusBadRequest, Err("数据不能为空"))
		return
	}

	payload := body
	var msg encryptedMessage
	if json.Unmarshal(body, &msg) == nil && msg.Encrypted && msg.Data != "" {
		plain, err := crypto.Decrypt(secret, msg.Data)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("解密失败"))
			return
		}
		payload = plain
	}

	var cfg gostConfig
	if err := json.Unmarshal(payload, &cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("解析失败"))
		return
	}

	s.cleanOrphanedServices(r, node.ID, cfg.Services)
	s.cleanOrphanedChains(r, node.ID, cfg.Chains)
	s.cleanOrphanedLimiters(r, node.ID, cfg.Limiters)

	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleFlowUpload(w http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("secret")
	if secret == "" {
		writeJSON(w, http.StatusUnauthorized, Err("缺少secret"))
		return
	}

	_, err := s.store.GetNodeBySecret(r.Context(), secret)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Err("节点不存在"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("读取失败"))
		return
	}
	if len(body) == 0 {
		writeJSON(w, http.StatusBadRequest, Err("数据不能为空"))
		return
	}

	payload := body
	var msg encryptedMessage
	if json.Unmarshal(body, &msg) == nil && msg.Encrypted && msg.Data != "" {
		plain, err := crypto.Decrypt(secret, msg.Data)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("解密失败"))
			return
		}
		payload = plain
	}

	var dto flowDTO
	if err := json.Unmarshal(payload, &dto); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("解析失败"))
		return
	}
	if dto.N == "web_api" {
		_, _ = w.Write([]byte("ok"))
		return
	}

	parts := strings.Split(dto.N, "_")
	if len(parts) < 3 {
		writeJSON(w, http.StatusBadRequest, Err("服务名非法"))
		return
	}
	forwardID, _ := strconv.ParseInt(parts[0], 10, 64)
	userID, _ := strconv.ParseInt(parts[1], 10, 64)
	userTunnelID, _ := strconv.ParseInt(parts[2], 10, 64)

	if err := s.flow.Apply(r.Context(), flow.Update{
		ForwardID:    forwardID,
		UserID:       userID,
		UserTunnelID: userTunnelID,
		Down:         dto.D,
		Up:           dto.U,
	}); err != nil {
		writeJSON(w, http.StatusBadRequest, Err(err.Error()))
		return
	}

	s.checkAndPauseIfNeeded(r, forwardID, userID, userTunnelID)

	_, _ = w.Write([]byte("ok"))
}

func (s *Server) cleanOrphanedServices(r *http.Request, nodeID int64, services []configItem) {
	for _, svc := range services {
		if svc.Name == "" || svc.Name == "web_api" {
			continue
		}
		parts := strings.Split(svc.Name, "_")
		if len(parts) < 4 {
			continue
		}
		forwardID, _ := strconv.ParseInt(parts[0], 10, 64)
		base := strings.Join(parts[:3], "_")
		typ := parts[3]

		if _, err := s.store.GetForwardByID(r.Context(), forwardID); err == nil {
			continue
		}

		if typ == "tcp" {
			_ = s.enqueueGost(r, nodeID, "DeleteService", gost.DeleteServiceData(base))
		}
		if typ == "tls" {
			_ = s.enqueueGost(r, nodeID, "DeleteService", gost.DeleteRemoteServiceData(base))
		}
	}
}

func (s *Server) cleanOrphanedChains(r *http.Request, nodeID int64, chains []configItem) {
	for _, chain := range chains {
		if chain.Name == "" {
			continue
		}
		parts := strings.Split(chain.Name, "_")
		if len(parts) < 4 {
			continue
		}
		forwardID, _ := strconv.ParseInt(parts[0], 10, 64)
		base := strings.Join(parts[:3], "_")
		typ := parts[3]
		if typ != "chains" {
			continue
		}
		if _, err := s.store.GetForwardByID(r.Context(), forwardID); err == nil {
			continue
		}
		_ = s.enqueueGost(r, nodeID, "DeleteChains", gost.DeleteChainsData(base))
	}
}

func (s *Server) cleanOrphanedLimiters(r *http.Request, nodeID int64, limiters []configItem) {
	for _, limiter := range limiters {
		if limiter.Name == "" {
			continue
		}
		id, err := strconv.ParseInt(limiter.Name, 10, 64)
		if err != nil {
			continue
		}
		if _, err := s.store.GetSpeedLimitByID(r.Context(), id); err == nil {
			continue
		}
		_ = s.enqueueGost(r, nodeID, "DeleteLimiters", gost.DeleteLimitersData(id))
	}
}

func (s *Server) checkAndPauseIfNeeded(r *http.Request, forwardID, userID, userTunnelID int64) {
	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil {
		return
	}
	userFlowLimit := user.Flow * 1024 * 1024 * 1024
	userCurrent := user.InFlow + user.OutFlow
	if userFlowLimit < userCurrent || (user.ExpTime != 0 && user.ExpTime <= time.Now().UnixMilli()) || user.Status != 1 {
		s.pauseAllUserForwards(r, userID)
		return
	}

	if userTunnelID != 0 {
		ut, err := s.store.GetUserTunnelByID(r.Context(), userTunnelID)
		if err == nil {
			utLimit := ut.Flow * 1024 * 1024 * 1024
			utCurrent := ut.InFlow + ut.OutFlow
			if utLimit <= utCurrent || (ut.ExpTime != 0 && ut.ExpTime <= time.Now().UnixMilli()) || ut.Status != 1 {
				s.pauseSpecificForward(r, ut.UserID, ut.TunnelID)
				return
			}
		}
	}

	// forward status check
	forward, err := s.store.GetForwardByID(r.Context(), forwardID)
	if err == nil && forward.Status != 1 {
		s.pauseForwardByID(r, forwardID)
	}
}

func (s *Server) pauseAllUserForwards(r *http.Request, userID int64) {
	forwards, err := s.store.ListForwardsByUser(r.Context(), userID)
	if err != nil {
		return
	}
	for _, fw := range forwards {
		name := buildServiceName(fw.ID, fw.UserID, s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID))
		_ = s.enqueueGost(r, fw.InNodeID, "PauseService", gost.PauseServiceData(name))
		if fw.TunnelType == 2 {
			_ = s.enqueueGost(r, fw.OutNodeID, "PauseService", gost.PauseRemoteServiceData(name))
		}
		_ = s.store.UpdateForwardStatus(r.Context(), fw.ID, 0, "paused", time.Now().UnixMilli())
	}
}

func (s *Server) pauseSpecificForward(r *http.Request, userID, tunnelID int64) {
	forwards, err := s.store.ListForwardsByUser(r.Context(), userID)
	if err != nil {
		return
	}
	for _, fw := range forwards {
		if fw.TunnelID != tunnelID {
			continue
		}
		name := buildServiceName(fw.ID, fw.UserID, s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID))
		_ = s.enqueueGost(r, fw.InNodeID, "PauseService", gost.PauseServiceData(name))
		if fw.TunnelType == 2 {
			_ = s.enqueueGost(r, fw.OutNodeID, "PauseService", gost.PauseRemoteServiceData(name))
		}
		_ = s.store.UpdateForwardStatus(r.Context(), fw.ID, 0, "paused", time.Now().UnixMilli())
	}
}

func (s *Server) pauseForwardByID(r *http.Request, forwardID int64) {
	fw, err := s.store.GetForwardByID(r.Context(), forwardID)
	if err != nil {
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), fw.TunnelID)
	if err != nil {
		return
	}
	userTunnelID := s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID)
	name := buildServiceName(fw.ID, fw.UserID, userTunnelID)
	_ = s.enqueueGost(r, tunnel.InNodeID, "PauseService", gost.PauseServiceData(name))
	if tunnel.Type == 2 {
		_ = s.enqueueGost(r, tunnel.OutNodeID, "PauseService", gost.PauseRemoteServiceData(name))
	}
	_ = s.store.UpdateForwardStatus(r.Context(), fw.ID, 0, "paused", time.Now().UnixMilli())
}

func (s *Server) resolveUserTunnelID(r *http.Request, userID, tunnelID int64) int64 {
	return s.resolveUserTunnelIDCtx(r.Context(), userID, tunnelID)
}

func (s *Server) resolveUserTunnelIDCtx(ctx context.Context, userID, tunnelID int64) int64 {
	ut, err := s.store.GetUserTunnelByUserAndTunnel(ctx, userID, tunnelID)
	if err != nil {
		return 0
	}
	return ut.ID
}

func buildServiceName(forwardID, userID, userTunnelID int64) string {
	return strconv.FormatInt(forwardID, 10) + "_" + strconv.FormatInt(userID, 10) + "_" + strconv.FormatInt(userTunnelID, 10)
}
