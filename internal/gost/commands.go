package gost

import (
	"encoding/json"
	"strconv"
	"strings"
)

type TunnelConfig struct {
	Type          int64
	Protocol      string
	TCPListenAddr string
	UDPListenAddr string
}

func TcpPingData(ip string, port int) json.RawMessage {
	return mustJSON(map[string]any{
		"ip":      ip,
		"port":    port,
		"count":   4,
		"timeout": 5000,
	})
}

func AddLimitersData(name int64, speed int64) json.RawMessage {
	return mustJSON(map[string]any{
		"name":   int64ToString(name),
		"limits": []string{"$ " + int64ToString(speed) + "MB " + int64ToString(speed) + "MB"},
	})
}

func UpdateLimitersData(name int64, speed int64) json.RawMessage {
	return mustJSON(map[string]any{
		"limiter": int64ToString(name),
		"data": map[string]any{
			"name":   int64ToString(name),
			"limits": []string{"$ " + int64ToString(speed) + "MB " + int64ToString(speed) + "MB"},
		},
	})
}

func DeleteLimitersData(name int64) json.RawMessage {
	return mustJSON(map[string]any{
		"limiter": int64ToString(name),
	})
}

func AddServiceData(name string, inPort int64, limiter *int64, remoteAddr string, tunnel TunnelConfig, strategy string, interfaceName *string) json.RawMessage {
	services := []any{
		createServiceConfig(name, inPort, limiter, remoteAddr, "tcp", tunnel, strategy, interfaceName),
		createServiceConfig(name, inPort, limiter, remoteAddr, "udp", tunnel, strategy, interfaceName),
	}
	return mustJSON(services)
}

func UpdateServiceData(name string, inPort int64, limiter *int64, remoteAddr string, tunnel TunnelConfig, strategy string, interfaceName *string) json.RawMessage {
	services := []any{
		createServiceConfig(name, inPort, limiter, remoteAddr, "tcp", tunnel, strategy, interfaceName),
		createServiceConfig(name, inPort, limiter, remoteAddr, "udp", tunnel, strategy, interfaceName),
	}
	return mustJSON(services)
}

func DeleteServiceData(name string) json.RawMessage {
	return mustJSON(map[string]any{
		"services": []string{name + "_tcp", name + "_udp"},
	})
}

func AddRemoteServiceData(name string, outPort int64, remoteAddr string, protocol string, strategy string, interfaceName *string) json.RawMessage {
	data := map[string]any{
		"name":     name + "_tls",
		"addr":     ":" + int64ToString(outPort),
		"handler":  map[string]any{"type": "relay"},
		"listener": map[string]any{"type": protocol},
	}
	if interfaceName != nil && strings.TrimSpace(*interfaceName) != "" {
		data["metadata"] = map[string]any{"interface": *interfaceName}
	}
	data["forwarder"] = createForwarder(remoteAddr, strategy)
	return mustJSON([]any{data})
}

func UpdateRemoteServiceData(name string, outPort int64, remoteAddr string, protocol string, strategy string, interfaceName *string) json.RawMessage {
	return AddRemoteServiceData(name, outPort, remoteAddr, protocol, strategy, interfaceName)
}

func DeleteRemoteServiceData(name string) json.RawMessage {
	return mustJSON(map[string]any{
		"services": []string{name + "_tls"},
	})
}

func PauseServiceData(name string) json.RawMessage {
	return mustJSON(map[string]any{
		"services": []string{name + "_tcp", name + "_udp"},
	})
}

func ResumeServiceData(name string) json.RawMessage {
	return mustJSON(map[string]any{
		"services": []string{name + "_tcp", name + "_udp"},
	})
}

func PauseRemoteServiceData(name string) json.RawMessage {
	return mustJSON(map[string]any{
		"services": []string{name + "_tls"},
	})
}

func ResumeRemoteServiceData(name string) json.RawMessage {
	return mustJSON(map[string]any{
		"services": []string{name + "_tls"},
	})
}

func AddChainsData(name string, remoteAddr string, protocol string, interfaceName *string) json.RawMessage {
	data := buildChainsData(name, remoteAddr, protocol, interfaceName)
	return mustJSON(data)
}

func UpdateChainsData(name string, remoteAddr string, protocol string, interfaceName *string) json.RawMessage {
	data := buildChainsData(name, remoteAddr, protocol, interfaceName)
	return mustJSON(map[string]any{
		"chain": name + "_chains",
		"data":  data,
	})
}

func DeleteChainsData(name string) json.RawMessage {
	return mustJSON(map[string]any{"chain": name + "_chains"})
}

func buildChainsData(name string, remoteAddr string, protocol string, interfaceName *string) map[string]any {
	dialer := map[string]any{"type": protocol}
	if protocol == "quic" {
		dialer["metadata"] = map[string]any{"keepAlive": true, "ttl": "10s"}
	}

	node := map[string]any{
		"name":      "node-" + name,
		"addr":      remoteAddr,
		"connector": map[string]any{"type": "relay"},
		"dialer":    dialer,
	}
	if interfaceName != nil && strings.TrimSpace(*interfaceName) != "" {
		node["interface"] = *interfaceName
	}

	hop := map[string]any{
		"name":  "hop-" + name,
		"nodes": []any{node},
	}

	return map[string]any{
		"name": name + "_chains",
		"hops": []any{hop},
	}
}

func createServiceConfig(name string, inPort int64, limiter *int64, remoteAddr string, protocol string, tunnel TunnelConfig, strategy string, interfaceName *string) map[string]any {
	service := map[string]any{
		"name": name + "_" + protocol,
	}
	if protocol == "tcp" {
		service["addr"] = tunnel.TCPListenAddr + ":" + int64ToString(inPort)
	} else {
		service["addr"] = tunnel.UDPListenAddr + ":" + int64ToString(inPort)
	}
	if interfaceName != nil && strings.TrimSpace(*interfaceName) != "" {
		service["metadata"] = map[string]any{"interface": *interfaceName}
	}
	if limiter != nil {
		service["limiter"] = int64ToString(*limiter)
	}
	service["handler"] = createHandler(protocol, name, tunnel.Type)
	service["listener"] = createListener(protocol)
	if tunnel.Type == 1 {
		service["forwarder"] = createForwarder(remoteAddr, strategy)
	}
	return service
}

func createHandler(protocol string, name string, tunnelType int64) map[string]any {
	h := map[string]any{"type": protocol}
	if tunnelType != 1 {
		h["chain"] = name + "_chains"
	}
	return h
}

func createListener(protocol string) map[string]any {
	listener := map[string]any{"type": protocol}
	if protocol == "udp" {
		listener["metadata"] = map[string]any{"keepAlive": true}
	}
	return listener
}

func createForwarder(remoteAddr string, strategy string) map[string]any {
	parts := strings.Split(remoteAddr, ",")
	nodes := make([]any, 0, len(parts))
	for idx, addr := range parts {
		nodes = append(nodes, map[string]any{"name": "node_" + int64ToString(int64(idx+1)), "addr": strings.TrimSpace(addr)})
	}
	if strings.TrimSpace(strategy) == "" {
		strategy = "fifo"
	}
	return map[string]any{
		"nodes":    nodes,
		"selector": map[string]any{"strategy": strategy, "maxFails": 1, "failTimeout": "600s"},
	}
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
