package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"pixia-panel/internal/store"
)

type nodeCreateRequest struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	ServerIP string `json:"serverIp"`
	PortSta  int64  `json:"portSta"`
	PortEnd  int64  `json:"portEnd"`
}

type nodeUpdateRequest struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	ServerIP string `json:"serverIp"`
	PortSta  int64  `json:"portSta"`
	PortEnd  int64  `json:"portEnd"`
	HTTP     *int64 `json:"http"`
	TLS      *int64 `json:"tls"`
	Socks    *int64 `json:"socks"`
}

type nodeDeleteRequest struct {
	ID int64 `json:"id"`
}

type nodeInstallRequest struct {
	ID int64 `json:"id"`
}

type nodeCheckStatusRequest struct {
	NodeID *int64 `json:"nodeId"`
}

func (s *Server) handleNodeCreate(w http.ResponseWriter, r *http.Request) {
	var req nodeCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	secret := randomHex(16)
	node := &store.Node{
		Name:        req.Name,
		Secret:      secret,
		IP:          &req.IP,
		ServerIP:    req.ServerIP,
		PortSta:     req.PortSta,
		PortEnd:     req.PortEnd,
		HTTP:        0,
		TLS:         0,
		Socks:       0,
		CreatedTime: time.Now().UnixMilli(),
		Status:      0,
	}
	if _, err := s.store.InsertNode(r.Context(), node); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("创建失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("节点创建成功"))
}

func (s *Server) handleNodeList(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.store.ListNodes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK(nodes))
}

func (s *Server) handleNodeUpdate(w http.ResponseWriter, r *http.Request) {
	var req nodeUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	node, err := s.store.GetNodeByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("节点不存在"))
		return
	}

	if req.HTTP != nil {
		node.HTTP = *req.HTTP
	}
	if req.TLS != nil {
		node.TLS = *req.TLS
	}
	if req.Socks != nil {
		node.Socks = *req.Socks
	}

	node.Name = req.Name
	node.IP = &req.IP
	node.ServerIP = req.ServerIP
	node.PortSta = req.PortSta
	node.PortEnd = req.PortEnd
	node.UpdatedTime = ptrInt64(time.Now().UnixMilli())

	if err := s.store.UpdateNode(r.Context(), node); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("更新失败"))
		return
	}

	_ = s.store.UpdateTunnelsInIP(r.Context(), node.ID, req.IP)
	_ = s.store.UpdateTunnelsOutIP(r.Context(), node.ID, req.ServerIP)

	// notify node if online
	if node.Status == 1 {
		payload := map[string]any{
			"http":  node.HTTP,
			"tls":   node.TLS,
			"socks": node.Socks,
		}
		data, _ := json.Marshal(payload)
		_ = s.enqueueGost(r, node.ID, "SetProtocol", data)
	}

	writeJSON(w, http.StatusOK, OK("节点更新成功"))
}

func (s *Server) handleNodeDelete(w http.ResponseWriter, r *http.Request) {
	var req nodeDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	count, _ := s.store.CountTunnelsByInNode(r.Context(), req.ID)
	if count > 0 {
		writeJSON(w, http.StatusBadRequest, Err("该节点还有隧道作为入口使用"))
		return
	}
	count, _ = s.store.CountTunnelsByOutNode(r.Context(), req.ID)
	if count > 0 {
		writeJSON(w, http.StatusBadRequest, Err("该节点还有隧道作为出口使用"))
		return
	}
	if err := s.store.DeleteNode(r.Context(), req.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("删除失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("节点删除成功"))
}

func (s *Server) handleNodeInstall(w http.ResponseWriter, r *http.Request) {
	var req nodeInstallRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	node, err := s.store.GetNodeByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("节点不存在"))
		return
	}
	cfg, err := s.store.GetConfigByName(r.Context(), "ip")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("请先设置ip"))
		return
	}
	addr := formatServerAddr(cfg.Value)
	cmd := "curl -L https://github.com/bqlpfy/flux-panel/releases/download/1.4.3/install.sh -o ./install.sh && chmod +x ./install.sh && ./install.sh -a " + addr + " -s " + node.Secret
	writeJSON(w, http.StatusOK, OK(cmd))
}

func (s *Server) handleNodeCheckStatus(w http.ResponseWriter, r *http.Request) {
	var req nodeCheckStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.NodeID != nil && *req.NodeID != 0 {
		node, err := s.store.GetNodeByID(r.Context(), *req.NodeID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("节点不存在"))
			return
		}
		status := int64(0)
		if s.hub.Connected(node.ID) {
			status = 1
		}
		writeJSON(w, http.StatusOK, OK(map[string]any{
			"nodeId": node.ID,
			"status": status,
		}))
		return
	}

	nodes, err := s.store.ListNodes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	for i := range nodes {
		if s.hub.Connected(nodes[i].ID) {
			nodes[i].Status = 1
		} else {
			nodes[i].Status = 0
		}
	}
	writeJSON(w, http.StatusOK, OK(nodes))
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func formatServerAddr(addr string) string {
	if strings.HasPrefix(addr, "[") {
		return addr
	}
	if strings.Count(addr, ":") >= 2 { // ipv6 without brackets
		return "[" + addr + "]"
	}
	return addr
}
