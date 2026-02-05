package httpapi

import (
	"net/http"
	"time"

	"pixia-panel/internal/gost"
	"pixia-panel/internal/store"
)

type speedLimitCreateRequest struct {
	Name       string `json:"name"`
	Speed      int64  `json:"speed"`
	TunnelID   int64  `json:"tunnelId"`
	TunnelName string `json:"tunnelName"`
	Status     *int64 `json:"status"`
}

type speedLimitUpdateRequest struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Speed      int64  `json:"speed"`
	TunnelID   int64  `json:"tunnelId"`
	TunnelName string `json:"tunnelName"`
	Status     *int64 `json:"status"`
}

type speedLimitDeleteRequest struct {
	ID int64 `json:"id"`
}

func (s *Server) handleSpeedLimitCreate(w http.ResponseWriter, r *http.Request) {
	var req speedLimitCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), req.TunnelID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	limit := &store.SpeedLimit{
		Name:        req.Name,
		Speed:       req.Speed,
		TunnelID:    req.TunnelID,
		TunnelName:  req.TunnelName,
		CreatedTime: time.Now().UnixMilli(),
		Status:      1,
	}
	if req.Status != nil {
		limit.Status = *req.Status
	}
	id, err := s.store.InsertSpeedLimit(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("创建失败"))
		return
	}
	limit.ID = id

	data := gost.AddLimitersData(limit.ID, limit.Speed)
	_ = s.enqueueGost(r, tunnel.InNodeID, "AddLimiters", data)

	writeJSON(w, http.StatusOK, OK("ok"))
}

func (s *Server) handleSpeedLimitList(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListSpeedLimits(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK(list))
}

func (s *Server) handleSpeedLimitUpdate(w http.ResponseWriter, r *http.Request) {
	var req speedLimitUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	limit, err := s.store.GetSpeedLimitByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("限速规则不存在"))
		return
	}
	count, _ := s.store.CountUserTunnelsBySpeed(r.Context(), limit.ID)
	if count > 0 {
		writeJSON(w, http.StatusBadRequest, Err("该限速规则还有用户在使用"))
		return
	}
	limit.Name = req.Name
	limit.Speed = req.Speed
	limit.TunnelID = req.TunnelID
	limit.TunnelName = req.TunnelName
	if req.Status != nil {
		limit.Status = *req.Status
	}
	limit.UpdatedTime = ptrInt64(time.Now().UnixMilli())
	if err := s.store.UpdateSpeedLimit(r.Context(), limit); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("更新失败"))
		return
	}
	if tunnel, err := s.store.GetTunnelByID(r.Context(), req.TunnelID); err == nil {
		data := gost.UpdateLimitersData(limit.ID, limit.Speed)
		_ = s.enqueueGost(r, tunnel.InNodeID, "UpdateLimiters", data)
	}
	writeJSON(w, http.StatusOK, OK("限速规则更新成功"))
}

func (s *Server) handleSpeedLimitDelete(w http.ResponseWriter, r *http.Request) {
	var req speedLimitDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	limit, err := s.store.GetSpeedLimitByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("限速规则不存在"))
		return
	}
	if tunnel, err := s.store.GetTunnelByID(r.Context(), limit.TunnelID); err == nil {
		data := gost.DeleteLimitersData(limit.ID)
		_ = s.enqueueGost(r, tunnel.InNodeID, "DeleteLimiters", data)
	}
	_ = s.store.DeleteSpeedLimit(r.Context(), req.ID)
	writeJSON(w, http.StatusOK, OK("限速规则删除成功"))
}

func (s *Server) handleSpeedLimitTunnels(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListTunnels(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK(list))
}
