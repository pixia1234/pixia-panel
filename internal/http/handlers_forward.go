package httpapi

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pixia-panel/internal/gost"
	"pixia-panel/internal/store"
)

type forwardCreateRequest struct {
	Name          string  `json:"name"`
	TunnelID      int64   `json:"tunnelId"`
	RemoteAddr    string  `json:"remoteAddr"`
	Strategy      string  `json:"strategy"`
	InPort        *int64  `json:"inPort"`
	InterfaceName *string `json:"interfaceName"`
}

type forwardUpdateRequest struct {
	ID            int64   `json:"id"`
	UserID        int64   `json:"userId"`
	Name          string  `json:"name"`
	TunnelID      int64   `json:"tunnelId"`
	RemoteAddr    string  `json:"remoteAddr"`
	Strategy      string  `json:"strategy"`
	InPort        *int64  `json:"inPort"`
	InterfaceName *string `json:"interfaceName"`
}

type forwardDeleteRequest struct {
	ID int64 `json:"id"`
}

type forwardDiagnoseRequest struct {
	ForwardID int64 `json:"forwardId"`
}

type forwardUpdateOrderRequest struct {
	Forwards []struct {
		ID  int64 `json:"id"`
		Inx int64 `json:"inx"`
	} `json:"forwards"`
}

func (s *Server) handleForwardCreate(w http.ResponseWriter, r *http.Request) {
	var req forwardCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.Name == "" || req.RemoteAddr == "" {
		writeJSON(w, http.StatusBadRequest, Err("转发名称或远程地址不能为空"))
		return
	}
	if strings.TrimSpace(req.Strategy) == "" {
		req.Strategy = "fifo"
	}
	currentUserID := userIDFromCtx(r)
	roleID := roleIDFromCtx(r)

	user, err := s.store.GetUserByID(r.Context(), currentUserID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("用户不存在"))
		return
	}

	if roleID != 0 {
		if user.Status == 0 || (user.ExpTime != 0 && user.ExpTime <= time.Now().UnixMilli()) {
			writeJSON(w, http.StatusBadRequest, Err("用户已到期或被禁用"))
			return
		}
	}

	tunnel, err := s.store.GetTunnelByID(r.Context(), req.TunnelID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	if tunnel.Status != 1 {
		writeJSON(w, http.StatusBadRequest, Err("隧道已禁用"))
		return
	}

	var userTunnel *store.UserTunnel
	if roleID != 0 {
		ut, err := s.store.GetUserTunnelByUserAndTunnel(r.Context(), currentUserID, req.TunnelID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("用户没有该隧道权限"))
			return
		}
		if ut.Status != 1 || (ut.ExpTime != 0 && ut.ExpTime <= time.Now().UnixMilli()) {
			writeJSON(w, http.StatusBadRequest, Err("隧道权限已到期或被禁用"))
			return
		}
		userTunnel = ut

		if user.Flow*1024*1024*1024 <= user.InFlow+user.OutFlow {
			writeJSON(w, http.StatusBadRequest, Err("用户流量已用完"))
			return
		}
		if userTunnel.Flow*1024*1024*1024 <= userTunnel.InFlow+userTunnel.OutFlow {
			writeJSON(w, http.StatusBadRequest, Err("隧道流量已用完"))
			return
		}

		count, _ := s.store.CountForwardsByUser(r.Context(), currentUserID)
		if count >= user.Num {
			writeJSON(w, http.StatusBadRequest, Err("用户转发数量已满"))
			return
		}
		count, _ = s.store.CountForwardsByUserTunnel(r.Context(), currentUserID, req.TunnelID)
		if count >= userTunnel.Num {
			writeJSON(w, http.StatusBadRequest, Err("隧道转发数量已满"))
			return
		}
	}

	inPort, outPort, err := s.allocatePorts(r, tunnel, req.InPort, nil)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err(err.Error()))
		return
	}

	fw := &store.Forward{
		UserID:        currentUserID,
		UserName:      user.User,
		Name:          req.Name,
		TunnelID:      req.TunnelID,
		InPort:        inPort,
		OutPort:       outPort,
		RemoteAddr:    req.RemoteAddr,
		Strategy:      req.Strategy,
		InterfaceName: req.InterfaceName,
		InFlow:        0,
		OutFlow:       0,
		CreatedTime:   time.Now().UnixMilli(),
		UpdatedTime:   time.Now().UnixMilli(),
		Status:        1,
		Inx:           0,
		Lifecycle:     "creating",
	}

	id, err := s.store.InsertForward(r.Context(), fw)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("创建失败"))
		return
	}
	fw.ID = id

	limiter := (*int64)(nil)
	if userTunnel != nil {
		limiter = userTunnel.SpeedID
	}
	s.enqueueForwardGost(r, fw, tunnel, limiter, "AddService")

	writeJSON(w, http.StatusOK, OK("ok"))
}

func (s *Server) handleForwardList(w http.ResponseWriter, r *http.Request) {
	role := roleIDFromCtx(r)
	userID := userIDFromCtx(r)
	var list []store.ForwardWithTunnel
	var err error
	if role == 0 {
		list, err = s.store.ListForwardsAll(r.Context())
	} else {
		list, err = s.store.ListForwardsByUser(r.Context(), userID)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK(list))
}

func (s *Server) handleForwardUpdate(w http.ResponseWriter, r *http.Request) {
	var req forwardUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.Name == "" || req.RemoteAddr == "" {
		writeJSON(w, http.StatusBadRequest, Err("转发名称或远程地址不能为空"))
		return
	}
	if strings.TrimSpace(req.Strategy) == "" {
		req.Strategy = "fifo"
	}
	role := roleIDFromCtx(r)
	currentUserID := userIDFromCtx(r)

	fw, err := s.store.GetForwardByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("转发不存在"))
		return
	}

	if role != 0 && fw.UserID != currentUserID {
		writeJSON(w, http.StatusForbidden, Err("无权限"))
		return
	}

	tunnel, err := s.store.GetTunnelByID(r.Context(), req.TunnelID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	if tunnel.Status != 1 {
		writeJSON(w, http.StatusBadRequest, Err("隧道已禁用"))
		return
	}

	var inPort = fw.InPort
	var outPort = fw.OutPort
	if req.TunnelID != fw.TunnelID || (req.InPort != nil && *req.InPort != fw.InPort) {
		in, out, err := s.allocatePorts(r, tunnel, req.InPort, &fw.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err(err.Error()))
			return
		}
		inPort = in
		outPort = out
	}

	fw.Name = req.Name
	fw.TunnelID = req.TunnelID
	fw.RemoteAddr = req.RemoteAddr
	fw.Strategy = req.Strategy
	fw.InPort = inPort
	fw.OutPort = outPort
	fw.InterfaceName = req.InterfaceName
	fw.UpdatedTime = time.Now().UnixMilli()
	fw.Lifecycle = "updating"

	if err := s.store.UpdateForward(r.Context(), fw); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("更新失败"))
		return
	}

	limiter := s.resolveSpeedLimiter(r, fw.UserID, fw.TunnelID)
	s.enqueueForwardGost(r, fw, tunnel, limiter, "UpdateService")

	writeJSON(w, http.StatusOK, OK("端口转发更新成功"))
}

func (s *Server) handleForwardDelete(w http.ResponseWriter, r *http.Request) {
	var req forwardDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	fw, err := s.store.GetForwardByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("转发不存在"))
		return
	}
	role := roleIDFromCtx(r)
	if role != 0 && fw.UserID != userIDFromCtx(r) {
		writeJSON(w, http.StatusForbidden, Err("无权限"))
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), fw.TunnelID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}

	name := buildServiceName(fw.ID, fw.UserID, s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID))
	_ = s.enqueueGost(r, tunnel.InNodeID, "DeleteService", gost.DeleteServiceData(name))
	if tunnel.Type == 2 {
		_ = s.enqueueGost(r, tunnel.InNodeID, "DeleteChains", gost.DeleteChainsData(name))
		_ = s.enqueueGost(r, tunnel.OutNodeID, "DeleteService", gost.DeleteRemoteServiceData(name))
	}
	_ = s.store.DeleteForward(r.Context(), fw.ID)
	writeJSON(w, http.StatusOK, OK("端口转发删除成功"))
}

func (s *Server) handleForwardForceDelete(w http.ResponseWriter, r *http.Request) {
	var req forwardDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	fw, err := s.store.GetForwardByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("转发不存在"))
		return
	}
	role := roleIDFromCtx(r)
	if role != 0 && fw.UserID != userIDFromCtx(r) {
		writeJSON(w, http.StatusForbidden, Err("无权限"))
		return
	}
	_ = s.store.DeleteForward(r.Context(), req.ID)
	writeJSON(w, http.StatusOK, OK("端口转发强制删除成功"))
}

func (s *Server) handleForwardPause(w http.ResponseWriter, r *http.Request) {
	var req forwardDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	fw, err := s.store.GetForwardByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("转发不存在"))
		return
	}
	role := roleIDFromCtx(r)
	if role != 0 && fw.UserID != userIDFromCtx(r) {
		writeJSON(w, http.StatusForbidden, Err("无权限"))
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), fw.TunnelID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	name := buildServiceName(fw.ID, fw.UserID, s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID))
	_ = s.enqueueGost(r, tunnel.InNodeID, "PauseService", gost.PauseServiceData(name))
	if tunnel.Type == 2 {
		_ = s.enqueueGost(r, tunnel.OutNodeID, "PauseService", gost.PauseRemoteServiceData(name))
	}
	_ = s.store.UpdateForwardStatus(r.Context(), fw.ID, 0, "paused", time.Now().UnixMilli())
	writeJSON(w, http.StatusOK, OK("服务已暂停"))
}

func (s *Server) handleForwardResume(w http.ResponseWriter, r *http.Request) {
	var req forwardDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	fw, err := s.store.GetForwardByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("转发不存在"))
		return
	}
	role := roleIDFromCtx(r)
	if role != 0 && fw.UserID != userIDFromCtx(r) {
		writeJSON(w, http.StatusForbidden, Err("无权限"))
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), fw.TunnelID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	name := buildServiceName(fw.ID, fw.UserID, s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID))
	_ = s.enqueueGost(r, tunnel.InNodeID, "ResumeService", gost.ResumeServiceData(name))
	if tunnel.Type == 2 {
		_ = s.enqueueGost(r, tunnel.OutNodeID, "ResumeService", gost.ResumeRemoteServiceData(name))
	}
	_ = s.store.UpdateForwardStatus(r.Context(), fw.ID, 1, "active", time.Now().UnixMilli())
	writeJSON(w, http.StatusOK, OK("服务已恢复"))
}

func (s *Server) handleForwardDiagnose(w http.ResponseWriter, r *http.Request) {
	var req forwardDiagnoseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	fw, err := s.store.GetForwardByID(r.Context(), req.ForwardID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("转发不存在"))
		return
	}
	role := roleIDFromCtx(r)
	if role != 0 && fw.UserID != userIDFromCtx(r) {
		writeJSON(w, http.StatusForbidden, Err("无权限"))
		return
	}
	tunnel, err := s.store.GetTunnelByID(r.Context(), fw.TunnelID)
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
	remoteAddresses := strings.Split(fw.RemoteAddr, ",")
	if tunnel.Type == 1 {
		for _, addr := range remoteAddresses {
			host, port, err := parseTargetAddr(addr)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, Err("无法解析目标地址"))
				return
			}
			results = append(results, s.tcpPing(r.Context(), inNode, host, port, "转发->目标"))
		}
	} else {
		outNode, err := s.store.GetNodeByID(r.Context(), tunnel.OutNodeID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("出口节点不存在"))
			return
		}
		if fw.OutPort == nil {
			writeJSON(w, http.StatusBadRequest, Err("出口端口不存在"))
			return
		}
		results = append(results, s.tcpPing(r.Context(), inNode, outNode.ServerIP, int(*fw.OutPort), "入口->出口"))
		for _, addr := range remoteAddresses {
			host, port, err := parseTargetAddr(addr)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, Err("无法解析目标地址"))
				return
			}
			results = append(results, s.tcpPing(r.Context(), outNode, host, port, "出口->目标"))
		}
	}

	report := map[string]any{
		"forwardId":   req.ForwardID,
		"forwardName": fw.Name,
		"tunnelType":  map[bool]string{true: "端口转发", false: "隧道转发"}[tunnel.Type == 1],
		"results":     results,
		"timestamp":   time.Now().UnixMilli(),
	}
	writeJSON(w, http.StatusOK, OK(report))
}

func (s *Server) handleForwardUpdateOrder(w http.ResponseWriter, r *http.Request) {
	var req forwardUpdateOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	for _, fw := range req.Forwards {
		_ = s.store.UpdateForwardOrder(r.Context(), fw.ID, fw.Inx)
	}
	writeJSON(w, http.StatusOK, OK("更新成功"))
}

func (s *Server) allocatePorts(r *http.Request, tunnel *store.Tunnel, specifiedIn *int64, excludeID *int64) (int64, *int64, error) {
	inPort := int64(0)
	if specifiedIn != nil {
		if !s.isInPortAvailable(r, tunnel, *specifiedIn, excludeID) {
			return 0, nil, fmt.Errorf("指定的入口端口已被占用或不在允许范围内")
		}
		inPort = *specifiedIn
	} else {
		p, err := s.allocatePortForNode(r, tunnel.InNodeID, excludeID)
		if err != nil {
			return 0, nil, err
		}
		inPort = p
	}

	var outPort *int64
	if tunnel.Type == 2 {
		p, err := s.allocatePortForNode(r, tunnel.OutNodeID, excludeID)
		if err != nil {
			return 0, nil, err
		}
		outPort = &p
	}
	return inPort, outPort, nil
}

func (s *Server) isInPortAvailable(r *http.Request, tunnel *store.Tunnel, port int64, excludeID *int64) bool {
	node, err := s.store.GetNodeByID(r.Context(), tunnel.InNodeID)
	if err != nil {
		return false
	}
	if port < node.PortSta || port > node.PortEnd {
		return false
	}
	used, _ := s.listUsedPortsOnNode(r, tunnel.InNodeID, excludeID)
	_, ok := used[port]
	return !ok
}

func (s *Server) allocatePortForNode(r *http.Request, nodeID int64, excludeID *int64) (int64, error) {
	node, err := s.store.GetNodeByID(r.Context(), nodeID)
	if err != nil {
		return 0, fmt.Errorf("节点不存在")
	}
	used, _ := s.listUsedPortsOnNode(r, nodeID, excludeID)
	for port := node.PortSta; port <= node.PortEnd; port++ {
		if _, ok := used[port]; !ok {
			return port, nil
		}
	}
	return 0, fmt.Errorf("端口已满")
}

func (s *Server) listUsedPortsOnNode(r *http.Request, nodeID int64, excludeID *int64) (map[int64]struct{}, error) {
	used := make(map[int64]struct{})
	exclude := int64(0)
	if excludeID != nil {
		exclude = *excludeID
	}

	// in ports
	query := `SELECT f.in_port FROM forward f JOIN tunnel t ON f.tunnel_id = t.id WHERE t.in_node_id = ?`
	args := []any{nodeID}
	if exclude != 0 {
		query += " AND f.id != ?"
		args = append(args, exclude)
	}
	rows, err := s.store.DB().QueryContext(r.Context(), query, args...)
	if err == nil {
		for rows.Next() {
			var port sql.NullInt64
			if err := rows.Scan(&port); err == nil && port.Valid {
				used[port.Int64] = struct{}{}
			}
		}
		_ = rows.Close()
	}

	// out ports
	query = `SELECT f.out_port FROM forward f JOIN tunnel t ON f.tunnel_id = t.id WHERE t.out_node_id = ? AND f.out_port IS NOT NULL`
	args = []any{nodeID}
	if exclude != 0 {
		query += " AND f.id != ?"
		args = append(args, exclude)
	}
	rows, err = s.store.DB().QueryContext(r.Context(), query, args...)
	if err == nil {
		for rows.Next() {
			var port sql.NullInt64
			if err := rows.Scan(&port); err == nil && port.Valid {
				used[port.Int64] = struct{}{}
			}
		}
		_ = rows.Close()
	}

	return used, nil
}

func (s *Server) enqueueForwardGost(r *http.Request, fw *store.Forward, tunnel *store.Tunnel, limiter *int64, action string) {
	name := buildServiceName(fw.ID, fw.UserID, s.resolveUserTunnelID(r, fw.UserID, fw.TunnelID))
	data := gost.AddServiceData(name, fw.InPort, limiter, fw.RemoteAddr, gost.TunnelConfig{Type: tunnel.Type, Protocol: tunnel.Protocol, TCPListenAddr: tunnel.TCPListenAddr, UDPListenAddr: tunnel.UDPListenAddr}, fw.Strategy, fw.InterfaceName)
	if action == "UpdateService" {
		data = gost.UpdateServiceData(name, fw.InPort, limiter, fw.RemoteAddr, gost.TunnelConfig{Type: tunnel.Type, Protocol: tunnel.Protocol, TCPListenAddr: tunnel.TCPListenAddr, UDPListenAddr: tunnel.UDPListenAddr}, fw.Strategy, fw.InterfaceName)
	}
	_ = s.enqueueGost(r, tunnel.InNodeID, action, data)

	if tunnel.Type == 2 && fw.OutPort != nil {
		remote := gost.AddRemoteServiceData(name, *fw.OutPort, fw.RemoteAddr, tunnel.Protocol, fw.Strategy, fw.InterfaceName)
		if action == "UpdateService" {
			remote = gost.UpdateRemoteServiceData(name, *fw.OutPort, fw.RemoteAddr, tunnel.Protocol, fw.Strategy, fw.InterfaceName)
		}
		_ = s.enqueueGost(r, tunnel.OutNodeID, action, remote)
		chains := gost.AddChainsData(name, tunnel.OutIP+":"+strconv.FormatInt(*fw.OutPort, 10), tunnel.Protocol, fw.InterfaceName)
		if action == "UpdateService" {
			chains = gost.UpdateChainsData(name, tunnel.OutIP+":"+strconv.FormatInt(*fw.OutPort, 10), tunnel.Protocol, fw.InterfaceName)
		}
		_ = s.enqueueGost(r, tunnel.InNodeID, map[string]string{"AddService": "AddChains", "UpdateService": "UpdateChains"}[action], chains)
	}
}

func dialAddr(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
