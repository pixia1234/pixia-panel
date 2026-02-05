package gost

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"pixia-panel/internal/auth"
	"pixia-panel/internal/crypto"
)

var (
	ErrNodeNotConnected = errors.New("node not connected")
	ErrResponseTimeout  = errors.New("response timeout")
)

type Response struct {
	Type    string
	Success bool
	Message string
	Data    json.RawMessage
}

type Hub struct {
	mu      sync.RWMutex
	conns   map[int64]*websocket.Conn
	secrets map[int64]string

	adminMu sync.Mutex
	admins  map[*websocket.Conn]struct{}

	jwtSecret []byte

	pendingMu sync.Mutex
	pending   map[string]chan Response
}

func NewHub() *Hub {
	return &Hub{
		conns:   make(map[int64]*websocket.Conn),
		secrets: make(map[int64]string),
		admins:  make(map[*websocket.Conn]struct{}),
		pending: make(map[string]chan Response),
	}
}

func (h *Hub) Register(nodeID int64, conn *websocket.Conn, secret string) {
	h.mu.Lock()
	if old, ok := h.conns[nodeID]; ok && old != conn {
		_ = old.Close()
	}
	h.conns[nodeID] = conn
	h.secrets[nodeID] = secret
	h.mu.Unlock()
}

func (h *Hub) Unregister(nodeID int64) {
	h.mu.Lock()
	if conn, ok := h.conns[nodeID]; ok {
		_ = conn.Close()
		delete(h.conns, nodeID)
	}
	delete(h.secrets, nodeID)
	h.mu.Unlock()
}

func (h *Hub) Connected(nodeID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.conns[nodeID]
	return ok
}

func (h *Hub) SetJWTSecret(secret []byte) {
	h.jwtSecret = secret
}

func (h *Hub) Send(ctx context.Context, nodeID int64, action string, data json.RawMessage) error {
	msg := map[string]any{
		"type": action,
		"data": json.RawMessage(data),
	}
	return h.send(nodeID, msg)
}

func (h *Hub) SendAndWait(ctx context.Context, nodeID int64, action string, data json.RawMessage, timeout time.Duration) (Response, error) {
	reqID := randomID()
	ch := make(chan Response, 1)
	h.pendingMu.Lock()
	h.pending[reqID] = ch
	h.pendingMu.Unlock()

	msg := map[string]any{
		"type":      action,
		"data":      json.RawMessage(data),
		"requestId": reqID,
	}
	if err := h.send(nodeID, msg); err != nil {
		h.pendingMu.Lock()
		delete(h.pending, reqID)
		h.pendingMu.Unlock()
		return Response{}, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		h.pendingMu.Lock()
		delete(h.pending, reqID)
		h.pendingMu.Unlock()
		return Response{}, ctx.Err()
	case <-timer.C:
		h.pendingMu.Lock()
		delete(h.pending, reqID)
		h.pendingMu.Unlock()
		return Response{}, ErrResponseTimeout
	}
}

func (h *Hub) send(nodeID int64, msg map[string]any) error {
	h.mu.RLock()
	conn, ok := h.conns[nodeID]
	secret := h.secrets[nodeID]
	h.mu.RUnlock()
	if !ok {
		return ErrNodeNotConnected
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if secret != "" {
		if enc, err := crypto.Encrypt(secret, payload); err == nil {
			wrapper := map[string]any{
				"encrypted": true,
				"data":      enc,
				"timestamp": time.Now().UnixMilli(),
			}
			if wrapped, err := json.Marshal(wrapper); err == nil {
				payload = wrapped
			}
		}
	}

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, payload)
}

type NodeLookup interface {
	LookupBySecret(ctx context.Context, secret string) (int64, error)
}

type NodeStatusUpdater interface {
	UpdateNodeStatus(ctx context.Context, nodeID int64, status int64, version *string, http, tls, socks *int64) error
}

// ServeWS upgrades and registers a websocket connection for a node.
func (h *Hub) ServeWS(lookup NodeLookup) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	return func(w http.ResponseWriter, r *http.Request) {
		connType := r.URL.Query().Get("type")
		if connType == "0" {
			token := r.URL.Query().Get("secret")
			if token == "" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}
			if len(h.jwtSecret) > 0 {
				if _, err := auth.Parse(h.jwtSecret, token); err != nil {
					http.Error(w, "invalid token", http.StatusUnauthorized)
					return
				}
			}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			h.registerAdmin(conn)
			defer h.unregisterAdmin(conn)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}

		secret := r.URL.Query().Get("secret")
		if secret == "" {
			http.Error(w, "missing secret", http.StatusUnauthorized)
			return
		}

		nodeID, err := lookup.LookupBySecret(r.Context(), secret)
		if err != nil {
			http.Error(w, "invalid secret", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		h.Register(nodeID, conn, secret)
		h.updateNodeStatus(r, lookup, nodeID, 1)
		h.broadcastStatus(nodeID, 1)
		defer func() {
			h.updateNodeStatus(r, lookup, nodeID, 0)
			h.broadcastStatus(nodeID, 0)
			h.Unregister(nodeID)
		}()

		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			h.handleMessage(nodeID, secret, payload)
		}
	}
}

func (h *Hub) updateNodeStatus(r *http.Request, lookup NodeLookup, nodeID int64, status int64) {
	updater, ok := lookup.(NodeStatusUpdater)
	if !ok {
		return
	}
	var version *string
	var httpVal, tlsVal, socksVal *int64
	if status == 1 {
		if v := r.URL.Query().Get("version"); v != "" {
			version = &v
		}
		if v := parseQueryInt64(r, "http"); v != nil {
			httpVal = v
		}
		if v := parseQueryInt64(r, "tls"); v != nil {
			tlsVal = v
		}
		if v := parseQueryInt64(r, "socks"); v != nil {
			socksVal = v
		}
	}
	_ = updater.UpdateNodeStatus(r.Context(), nodeID, status, version, httpVal, tlsVal, socksVal)
}

func parseQueryInt64(r *http.Request, key string) *int64 {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	var n int64
	for i := 0; i < len(val); i++ {
		if val[i] < '0' || val[i] > '9' {
			return nil
		}
		n = n*10 + int64(val[i]-'0')
	}
	return &n
}

func (h *Hub) handleMessage(nodeID int64, secret string, payload []byte) {
	msg := payload
	var wrapper struct {
		Encrypted bool   `json:"encrypted"`
		Data      string `json:"data"`
	}
	if err := json.Unmarshal(payload, &wrapper); err == nil && wrapper.Encrypted && wrapper.Data != "" {
		if plain, err := crypto.Decrypt(secret, wrapper.Data); err == nil {
			msg = plain
		}
	}

	var sysInfo map[string]any
	if err := json.Unmarshal(msg, &sysInfo); err == nil {
		if _, ok := sysInfo["memory_usage"]; ok {
			h.broadcastInfo(nodeID, sysInfo)
			_ = h.send(nodeID, map[string]any{"type": "call"})
			return
		}
	}

	var resp struct {
		Type      string          `json:"type"`
		Success   bool            `json:"success"`
		Message   string          `json:"message"`
		Data      json.RawMessage `json:"data"`
		RequestId string          `json:"requestId"`
	}
	if err := json.Unmarshal(msg, &resp); err != nil {
		return
	}
	if resp.RequestId == "" {
		return
	}

	h.pendingMu.Lock()
	ch, ok := h.pending[resp.RequestId]
	if ok {
		delete(h.pending, resp.RequestId)
	}
	h.pendingMu.Unlock()
	if ok {
		ch <- Response{
			Type:    resp.Type,
			Success: resp.Success,
			Message: resp.Message,
			Data:    resp.Data,
		}
	}
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Hub) registerAdmin(conn *websocket.Conn) {
	h.adminMu.Lock()
	h.admins[conn] = struct{}{}
	h.adminMu.Unlock()
}

func (h *Hub) unregisterAdmin(conn *websocket.Conn) {
	h.adminMu.Lock()
	delete(h.admins, conn)
	h.adminMu.Unlock()
	_ = conn.Close()
}

func (h *Hub) broadcastStatus(nodeID int64, status int64) {
	h.broadcast(map[string]any{
		"id":   nodeID,
		"type": "status",
		"data": status,
	})
}

func (h *Hub) broadcastInfo(nodeID int64, info map[string]any) {
	h.broadcast(map[string]any{
		"id":   nodeID,
		"type": "info",
		"data": info,
	})
}

func (h *Hub) broadcast(payload map[string]any) {
	msg, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.adminMu.Lock()
	for conn := range h.admins {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			delete(h.admins, conn)
			_ = conn.Close()
		}
	}
	h.adminMu.Unlock()
}
