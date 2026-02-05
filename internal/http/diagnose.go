package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"pixia-panel/internal/gost"
	"pixia-panel/internal/store"
)

type diagnosisResult struct {
	NodeID      int64   `json:"nodeId"`
	NodeName    string  `json:"nodeName"`
	TargetIP    string  `json:"targetIp"`
	TargetPort  int     `json:"targetPort"`
	Description string  `json:"description"`
	Success     bool    `json:"success"`
	Message     string  `json:"message"`
	AverageTime float64 `json:"averageTime"`
	PacketLoss  float64 `json:"packetLoss"`
	Timestamp   int64   `json:"timestamp"`
}

type tcpPingResponse struct {
	Success      bool    `json:"success"`
	AverageTime  float64 `json:"averageTime"`
	PacketLoss   float64 `json:"packetLoss"`
	ErrorMessage string  `json:"errorMessage"`
}

func (s *Server) tcpPing(ctx context.Context, node *store.Node, targetIP string, port int, description string) diagnosisResult {
	result := diagnosisResult{
		TargetIP:    targetIP,
		TargetPort:  port,
		Description: description,
		Timestamp:   time.Now().UnixMilli(),
	}
	if node != nil {
		result.NodeID = node.ID
		result.NodeName = node.Name
	}

	if node == nil {
		result.Success = false
		result.Message = "节点不存在"
		result.AverageTime = -1
		result.PacketLoss = 100
		return result
	}

	resp, err := s.hub.SendAndWait(ctx, node.ID, "TcpPing", gost.TcpPingData(targetIP, port), 10*time.Second)
	if err != nil {
		result.Success = false
		result.Message = err.Error()
		result.AverageTime = -1
		result.PacketLoss = 100
		return result
	}

	result.Success = resp.Success
	if resp.Message != "" {
		result.Message = resp.Message
	} else if resp.Success {
		result.Message = "TCP连接成功"
	} else {
		result.Message = "TCP连接失败"
	}

	if len(resp.Data) > 0 {
		var data tcpPingResponse
		if err := json.Unmarshal(resp.Data, &data); err == nil {
			result.Success = data.Success
			if data.Success {
				result.Message = "TCP连接成功"
			} else if data.ErrorMessage != "" {
				result.Message = data.ErrorMessage
			}
			result.AverageTime = data.AverageTime
			result.PacketLoss = data.PacketLoss
			return result
		}
	}

	if result.Success {
		result.AverageTime = 0
		result.PacketLoss = 0
	} else {
		result.AverageTime = -1
		result.PacketLoss = 100
	}
	return result
}

func parseTargetAddr(raw string) (string, int, error) {
	addr := strings.TrimSpace(raw)
	if addr == "" {
		return "", 0, errors.New("empty address")
	}
	if strings.Contains(addr, "://") {
		u, err := url.Parse(addr)
		if err == nil && u.Host != "" {
			addr = u.Host
		}
	}
	if strings.Count(addr, ":") > 1 && !strings.HasPrefix(addr, "[") {
		addr = "[" + addr + "]"
	}
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	if host == "" || port <= 0 {
		return "", 0, errors.New("invalid address")
	}
	return host, port, nil
}
