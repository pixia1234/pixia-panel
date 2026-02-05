package httpapi

import (
	"net/http"
	"strings"

	"pixia-panel/internal/auth"
)

type configGetRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleConfigList(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListConfigs(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	cfg := make(map[string]string, len(list))
	includeSecret := s.isAdminRequest(r)
	for _, item := range list {
		if item.Name == "turnstile_secret_key" && !includeSecret {
			continue
		}
		cfg[item.Name] = item.Value
	}
	writeJSON(w, http.StatusOK, OK(cfg))
}

func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	var req configGetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.Name == "turnstile_secret_key" && !s.isAdminRequest(r) {
		writeJSON(w, http.StatusForbidden, Err("权限不足"))
		return
	}
	cfg, err := s.store.GetConfigByName(r.Context(), req.Name)
	if err != nil {
		writeJSON(w, http.StatusOK, OK(nil))
		return
	}
	writeJSON(w, http.StatusOK, OK(cfg))
}

func (s *Server) handleConfigUpdateBatch(w http.ResponseWriter, r *http.Request) {
	var payload map[string]string
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	for k, v := range payload {
		_ = s.store.UpsertConfig(r.Context(), k, v)
	}
	writeJSON(w, http.StatusOK, OK("更新成功"))
}

func (s *Server) handleConfigUpdateSingle(w http.ResponseWriter, r *http.Request) {
	var payload map[string]string
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	name := payload["name"]
	value := payload["value"]
	if name == "" {
		writeJSON(w, http.StatusBadRequest, Err("name不能为空"))
		return
	}
	_ = s.store.UpsertConfig(r.Context(), name, value)
	writeJSON(w, http.StatusOK, OK("更新成功"))
}

func (s *Server) isAdminRequest(r *http.Request) bool {
	if role, ok := r.Context().Value(ctxRoleID).(int64); ok {
		return role == 0
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}
	tokenStr := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	claims, err := auth.Parse(s.jwtSecret, tokenStr)
	if err != nil {
		return false
	}
	return claims.RoleID == 0
}
