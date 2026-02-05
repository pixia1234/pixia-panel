package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"pixia-panel/internal/gost"
	"pixia-panel/internal/store"
)

type tunnelCreateRequest struct {
	Name          string  `json:"name"`
	InNodeID      int64   `json:"inNodeId"`
	OutNodeID     *int64  `json:"outNodeId"`
	Type          int64   `json:"type"`
	Flow          int64   `json:"flow"`
	TrafficRatio  float64 `json:"trafficRatio"`
	InterfaceName *string `json:"interfaceName"`
	Protocol      string  `json:"protocol"`
	TCPListenAddr string  `json:"tcpListenAddr"`
	UDPListenAddr string  `json:"udpListenAddr"`
	Status        *int64  `json:"status"`
}

type tunnelUpdateRequest struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Type          *int64  `json:"type"`
	InNodeID      *int64  `json:"inNodeId"`
	OutNodeID     *int64  `json:"outNodeId"`
	Flow          int64   `json:"flow"`
	TrafficRatio  float64 `json:"trafficRatio"`
	Protocol      string  `json:"protocol"`
	TCPListenAddr string  `json:"tcpListenAddr"`
	UDPListenAddr string  `json:"udpListenAddr"`
	InterfaceName *string `json:"interfaceName"`
	Status        *int64  `json:"status"`
}

type tunnelDeleteRequest struct {
	ID int64 `json:"id"`
}

type tunnelGetRequest struct {
	ID int64 `json:"id"`
}

type userTunnelAssignRequest struct {
	UserID        int64  `json:"userId"`
	TunnelID      int64  `json:"tunnelId"`
	Flow          int64  `json:"flow"`
	Num           int64  `json:"num"`
	FlowResetTime int64  `json:"flowResetTime"`
	ExpTime       int64  `json:"expTime"`
	SpeedID       *int64 `json:"speedId"`
}

type userTunnelListRequest struct {
	UserID int64 `json:"userId"`
}

type userTunnelRemoveRequest struct {
	ID int64 `json:"id"`
}

type userTunnelUpdateRequest struct {
	ID            int64  `json:"id"`
	Flow          int64  `json:"flow"`
	Num           int64  `json:"num"`
	FlowResetTime int64  `json:"flowResetTime"`
	ExpTime       int64  `json:"expTime"`
	Status        int64  `json:"status"`
	SpeedID       *int64 `json:"speedId"`
}

type tunnelDiagnoseRequest struct {
	TunnelID int64 `json:"tunnelId"`
}

type tunnelView struct {
	store.Tunnel
	InNodePortSta int64 `json:"inNodePortSta,omitempty"`
	InNodePortEnd int64 `json:"inNodePortEnd,omitempty"`
}

func (s *Server) handleTunnelCreate(w http.ResponseWriter, r *http.Request) {
	var req tunnelCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, Err("隧道名称不能为空"))
		return
	}
	if _, err := s.store.GetTunnelByName(r.Context(), req.Name); err == nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道名称已存在"))
		return
	}
	if req.TrafficRatio == 0 {
		req.TrafficRatio = 1.0
	}
	if req.Protocol == "" {
		req.Protocol = "tls"
	}

	inNode, err := s.store.GetNodeByID(r.Context(), req.InNodeID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("入口节点不存在"))
		return
	}
	outNodeID := req.InNodeID
	if req.OutNodeID != nil {
		outNodeID = *req.OutNodeID
	}
	outNode, err := s.store.GetNodeByID(r.Context(), outNodeID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("出口节点不存在"))
		return
	}

	tunnel := &store.Tunnel{
		Name:          req.Name,
		TrafficRatio:  req.TrafficRatio,
		InNodeID:      req.InNodeID,
		InIP:          derefString(inNode.IP),
		OutNodeID:     outNodeID,
		OutIP:         outNode.ServerIP,
		Type:          req.Type,
		Protocol:      defaultString(req.Protocol, "tls"),
		Flow:          req.Flow,
		TCPListenAddr: defaultString(req.TCPListenAddr, "[::]"),
		UDPListenAddr: defaultString(req.UDPListenAddr, "[::]"),
		InterfaceName: req.InterfaceName,
		CreatedTime:   time.Now().UnixMilli(),
		UpdatedTime:   time.Now().UnixMilli(),
		Status:        1,
	}
	if req.Status != nil {
		tunnel.Status = *req.Status
	}

	if _, err := s.store.InsertTunnel(r.Context(), tunnel); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("创建失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("隧道创建成功"))
}

func (s *Server) handleTunnelList(w http.ResponseWriter, r *http.Request) {
	tunnels, err := s.store.ListTunnels(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	views := s.decorateTunnels(r.Context(), tunnels)
	writeJSON(w, http.StatusOK, OK(views))
}

func (s *Server) handleTunnelGet(w http.ResponseWriter, r *http.Request) {
	var req tunnelGetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	views := s.decorateTunnels(r.Context(), []store.Tunnel{*tunnel})
	if len(views) > 0 {
		writeJSON(w, http.StatusOK, OK(views[0]))
		return
	}
	writeJSON(w, http.StatusOK, OK(tunnel))
}

func (s *Server) handleTunnelUpdate(w http.ResponseWriter, r *http.Request) {
	var req tunnelUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	if req.TrafficRatio == 0 {
		req.TrafficRatio = tunnel.TrafficRatio
	}
	if req.Protocol == "" {
		req.Protocol = tunnel.Protocol
	}

	tunnel.Name = req.Name
	if req.Type != nil {
		tunnel.Type = *req.Type
	}
	if req.InNodeID != nil {
		inNode, err := s.store.GetNodeByID(r.Context(), *req.InNodeID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("入口节点不存在"))
			return
		}
		tunnel.InNodeID = *req.InNodeID
		tunnel.InIP = derefString(inNode.IP)
	}
	if req.OutNodeID != nil {
		outNode, err := s.store.GetNodeByID(r.Context(), *req.OutNodeID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("出口节点不存在"))
			return
		}
		tunnel.OutNodeID = *req.OutNodeID
		tunnel.OutIP = outNode.ServerIP
	}
	tunnel.Flow = req.Flow
	tunnel.TrafficRatio = req.TrafficRatio
	tunnel.Protocol = req.Protocol
	tunnel.TCPListenAddr = req.TCPListenAddr
	tunnel.UDPListenAddr = req.UDPListenAddr
	tunnel.InterfaceName = req.InterfaceName
	if req.Status != nil {
		tunnel.Status = *req.Status
	}
	tunnel.UpdatedTime = time.Now().UnixMilli()

	if err := s.store.UpdateTunnel(r.Context(), tunnel); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("更新失败"))
		return
	}

	// update related forwards on node
	forwards, _ := s.store.ListForwardsByTunnel(r.Context(), tunnel.ID)
	for _, fw := range forwards {
		userTunnelID := s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID)
		name := buildServiceName(fw.ID, fw.UserID, userTunnelID)
		limiter := s.resolveSpeedLimiter(r, fw.UserID, fw.TunnelID)
		data := gost.UpdateServiceData(name, fw.InPort, limiter, fw.RemoteAddr, gost.TunnelConfig{Type: tunnel.Type, Protocol: tunnel.Protocol, TCPListenAddr: tunnel.TCPListenAddr, UDPListenAddr: tunnel.UDPListenAddr}, fw.Strategy, fw.InterfaceName)
		_ = s.enqueueGost(r, tunnel.InNodeID, "UpdateService", data)
		if tunnel.Type == 2 && fw.OutPort != nil {
			remote := gost.UpdateRemoteServiceData(name, *fw.OutPort, fw.RemoteAddr, tunnel.Protocol, fw.Strategy, fw.InterfaceName)
			_ = s.enqueueGost(r, tunnel.OutNodeID, "UpdateService", remote)
			chains := gost.UpdateChainsData(name, tunnel.OutIP+":"+strconv.FormatInt(*fw.OutPort, 10), tunnel.Protocol, fw.InterfaceName)
			_ = s.enqueueGost(r, tunnel.InNodeID, "UpdateChains", chains)
		}
	}

	writeJSON(w, http.StatusOK, OK("隧道更新成功"))
}

func (s *Server) handleTunnelDelete(w http.ResponseWriter, r *http.Request) {
	var req tunnelDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	count, _ := s.store.CountForwardsByTunnel(r.Context(), req.ID)
	if count > 0 {
		writeJSON(w, http.StatusBadRequest, Err("该隧道还有转发在使用"))
		return
	}
	count, _ = s.store.CountUserTunnelsByTunnel(r.Context(), req.ID)
	if count > 0 {
		writeJSON(w, http.StatusBadRequest, Err("该隧道还有用户权限关联"))
		return
	}
	if err := s.store.DeleteTunnel(r.Context(), req.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("删除失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("隧道删除成功"))
}

func (s *Server) handleUserTunnelAssign(w http.ResponseWriter, r *http.Request) {
	var req userTunnelAssignRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if _, err := s.store.GetUserTunnelByUserAndTunnel(r.Context(), req.UserID, req.TunnelID); err == nil {
		writeJSON(w, http.StatusBadRequest, Err("该用户已拥有此隧道权限"))
		return
	}
	ut := &store.UserTunnel{
		UserID:        req.UserID,
		TunnelID:      req.TunnelID,
		SpeedID:       req.SpeedID,
		Num:           req.Num,
		Flow:          req.Flow,
		InFlow:        0,
		OutFlow:       0,
		FlowResetTime: req.FlowResetTime,
		ExpTime:       req.ExpTime,
		Status:        1,
	}
	if _, err := s.store.InsertUserTunnel(r.Context(), ut); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("分配失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("用户隧道权限分配成功"))
}

func (s *Server) handleUserTunnelList(w http.ResponseWriter, r *http.Request) {
	var req userTunnelListRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	list, err := s.store.ListUserTunnelsByUser(r.Context(), req.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK(list))
}

func (s *Server) handleUserTunnelRemove(w http.ResponseWriter, r *http.Request) {
	var req userTunnelRemoveRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	ut, err := s.store.GetUserTunnelByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("用户隧道权限不存在"))
		return
	}

	forwards, _ := s.store.ListForwardsByUser(r.Context(), ut.UserID)
	for _, fw := range forwards {
		if fw.TunnelID != ut.TunnelID {
			continue
		}
		name := buildServiceName(fw.ID, fw.UserID, ut.ID)
		_ = s.enqueueGost(r, fw.InNodeID, "DeleteService", gost.DeleteServiceData(name))
		if fw.TunnelType == 2 {
			_ = s.enqueueGost(r, fw.InNodeID, "DeleteChains", gost.DeleteChainsData(name))
			_ = s.enqueueGost(r, fw.OutNodeID, "DeleteService", gost.DeleteRemoteServiceData(name))
		}
		_ = s.store.DeleteForward(r.Context(), fw.ID)
	}

	if err := s.store.DeleteUserTunnel(r.Context(), ut.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("删除失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("用户隧道权限删除成功"))
}

func (s *Server) handleUserTunnelUpdate(w http.ResponseWriter, r *http.Request) {
	var req userTunnelUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	ut, err := s.store.GetUserTunnelByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("用户隧道权限不存在"))
		return
	}
	oldSpeed := ut.SpeedID
	ut.Flow = req.Flow
	ut.Num = req.Num
	ut.FlowResetTime = req.FlowResetTime
	ut.ExpTime = req.ExpTime
	ut.Status = req.Status
	ut.SpeedID = req.SpeedID
	if err := s.store.UpdateUserTunnel(r.Context(), ut); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("更新失败"))
		return
	}

	if !equalInt64Ptr(oldSpeed, ut.SpeedID) {
		// update forward limiters
		forwards, _ := s.store.ListForwardsByUser(r.Context(), ut.UserID)
		for _, fw := range forwards {
			if fw.TunnelID != ut.TunnelID {
				continue
			}
			tunnel, err := s.store.GetTunnelByID(r.Context(), fw.TunnelID)
			if err != nil {
				continue
			}
			name := buildServiceName(fw.ID, fw.UserID, ut.ID)
			data := gost.UpdateServiceData(name, fw.InPort, ut.SpeedID, fw.RemoteAddr, gost.TunnelConfig{Type: tunnel.Type, Protocol: tunnel.Protocol, TCPListenAddr: tunnel.TCPListenAddr, UDPListenAddr: tunnel.UDPListenAddr}, fw.Strategy, fw.InterfaceName)
			_ = s.enqueueGost(r, tunnel.InNodeID, "UpdateService", data)
		}
	}

	writeJSON(w, http.StatusOK, OK("用户隧道权限更新成功"))
}

func (s *Server) handleUserTunnelAvailable(w http.ResponseWriter, r *http.Request) {
	role := roleIDFromCtx(r)
	userID := userIDFromCtx(r)
	if role == 0 {
		tunnels, _ := s.store.ListTunnels(r.Context())
		writeJSON(w, http.StatusOK, OK(s.decorateTunnels(r.Context(), tunnels)))
		return
	}
	list, _ := s.store.ListUserTunnelsByUser(r.Context(), userID)
	var tunnels []map[string]any
	for _, ut := range list {
		inSta, inEnd := s.lookupTunnelPorts(r.Context(), ut.TunnelID)
		tunnels = append(tunnels, map[string]any{
			"id":            ut.TunnelID,
			"name":          ut.TunnelName,
			"type":          ut.TunnelType,
			"inNodePortSta": inSta,
			"inNodePortEnd": inEnd,
		})
	}
	writeJSON(w, http.StatusOK, OK(tunnels))
}

func (s *Server) handleTunnelDiagnose(w http.ResponseWriter, r *http.Request) {
	var req tunnelDiagnoseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), req.TunnelID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	inNode, err := s.store.GetNodeByID(r.Context(), tunnel.InNodeID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("入口节点不存在"))
		return
	}

	var results []diagnosisResult
	if tunnel.Type == 1 {
		results = append(results, s.tcpPing(r.Context(), inNode, "www.google.com", 443, "入口->外网"))
	} else {
		outNode, err := s.store.GetNodeByID(r.Context(), tunnel.OutNodeID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("出口节点不存在"))
			return
		}
		outPort := int64(22)
		forwards, _ := s.store.ListForwardsByTunnel(r.Context(), tunnel.ID)
		for _, fw := range forwards {
			if fw.Status == 1 && fw.OutPort != nil {
				outPort = *fw.OutPort
				break
			}
		}
		results = append(results, s.tcpPing(r.Context(), inNode, outNode.ServerIP, int(outPort), "入口->出口"))
		results = append(results, s.tcpPing(r.Context(), outNode, "www.google.com", 443, "出口->外网"))
	}

	report := map[string]any{
		"tunnelId":   req.TunnelID,
		"tunnelName": tunnel.Name,
		"tunnelType": map[bool]string{true: "端口转发", false: "隧道转发"}[tunnel.Type == 1],
		"results":    results,
		"timestamp":  time.Now().UnixMilli(),
	}
	writeJSON(w, http.StatusOK, OK(report))
}

func defaultString(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func derefString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (s *Server) resolveSpeedLimiter(r *http.Request, userID, tunnelID int64) *int64 {
	ut, err := s.store.GetUserTunnelByUserAndTunnel(r.Context(), userID, tunnelID)
	if err != nil {
		return nil
	}
	return ut.SpeedID
}

func (s *Server) decorateTunnels(ctx context.Context, tunnels []store.Tunnel) []tunnelView {
	nodePorts := s.loadNodePorts(ctx)
	views := make([]tunnelView, 0, len(tunnels))
	for _, t := range tunnels {
		view := tunnelView{Tunnel: t}
		if ports, ok := nodePorts[t.InNodeID]; ok {
			view.InNodePortSta = ports[0]
			view.InNodePortEnd = ports[1]
		}
		views = append(views, view)
	}
	return views
}

func (s *Server) loadNodePorts(ctx context.Context) map[int64][2]int64 {
	nodes, err := s.store.ListNodes(ctx)
	if err != nil {
		return map[int64][2]int64{}
	}
	res := make(map[int64][2]int64, len(nodes))
	for _, node := range nodes {
		res[node.ID] = [2]int64{node.PortSta, node.PortEnd}
	}
	return res
}

func (s *Server) lookupTunnelPorts(ctx context.Context, tunnelID int64) (int64, int64) {
	tunnel, err := s.store.GetTunnelByID(ctx, tunnelID)
	if err != nil {
		return 0, 0
	}
	node, err := s.store.GetNodeByID(ctx, tunnel.InNodeID)
	if err != nil {
		return 0, 0
	}
	return node.PortSta, node.PortEnd
}
