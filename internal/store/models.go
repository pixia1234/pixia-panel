package store

import "encoding/json"

// Core database models (minimal fields for now).

type User struct {
	ID            int64  `json:"id"`
	User          string `json:"user"`
	Pwd           string `json:"pwd"`
	RoleID        int64  `json:"roleId"`
	ExpTime       int64  `json:"expTime"`
	Flow          int64  `json:"flow"`
	InFlow        int64  `json:"inFlow"`
	OutFlow       int64  `json:"outFlow"`
	FlowResetTime int64  `json:"flowResetTime"`
	Num           int64  `json:"num"`
	CreatedTime   int64  `json:"createdTime"`
	UpdatedTime   *int64 `json:"updatedTime"`
	Status        int64  `json:"status"`
}

type Node struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Secret      string  `json:"secret"`
	IP          *string `json:"ip"`
	ServerIP    string  `json:"serverIp"`
	PortSta     int64   `json:"portSta"`
	PortEnd     int64   `json:"portEnd"`
	Version     *string `json:"version"`
	HTTP        int64   `json:"http"`
	TLS         int64   `json:"tls"`
	Socks       int64   `json:"socks"`
	CreatedTime int64   `json:"createdTime"`
	UpdatedTime *int64  `json:"updatedTime"`
	Status      int64   `json:"status"`
}

type Tunnel struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	TrafficRatio  float64 `json:"trafficRatio"`
	InNodeID      int64   `json:"inNodeId"`
	InIP          string  `json:"inIp"`
	OutNodeID     int64   `json:"outNodeId"`
	OutIP         string  `json:"outIp"`
	Type          int64   `json:"type"`
	Protocol      string  `json:"protocol"`
	Flow          int64   `json:"flow"`
	TCPListenAddr string  `json:"tcpListenAddr"`
	UDPListenAddr string  `json:"udpListenAddr"`
	InterfaceName *string `json:"interfaceName"`
	CreatedTime   int64   `json:"createdTime"`
	UpdatedTime   int64   `json:"updatedTime"`
	Status        int64   `json:"status"`
}

type SpeedLimit struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Speed       int64  `json:"speed"`
	TunnelID    int64  `json:"tunnelId"`
	TunnelName  string `json:"tunnelName"`
	CreatedTime int64  `json:"createdTime"`
	UpdatedTime *int64 `json:"updatedTime"`
	Status      int64  `json:"status"`
}

type UserTunnel struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"userId"`
	TunnelID      int64  `json:"tunnelId"`
	SpeedID       *int64 `json:"speedId"`
	Num           int64  `json:"num"`
	Flow          int64  `json:"flow"`
	InFlow        int64  `json:"inFlow"`
	OutFlow       int64  `json:"outFlow"`
	FlowResetTime int64  `json:"flowResetTime"`
	ExpTime       int64  `json:"expTime"`
	Status        int64  `json:"status"`
}

type Forward struct {
	ID            int64   `json:"id"`
	UserID        int64   `json:"userId"`
	UserName      string  `json:"userName"`
	Name          string  `json:"name"`
	TunnelID      int64   `json:"tunnelId"`
	InPort        int64   `json:"inPort"`
	OutPort       *int64  `json:"outPort"`
	RemoteAddr    string  `json:"remoteAddr"`
	Strategy      string  `json:"strategy"`
	InterfaceName *string `json:"interfaceName"`
	InFlow        int64   `json:"inFlow"`
	OutFlow       int64   `json:"outFlow"`
	CreatedTime   int64   `json:"createdTime"`
	UpdatedTime   int64   `json:"updatedTime"`
	Status        int64   `json:"status"`
	Inx           int64   `json:"inx"`
	Lifecycle     string  `json:"lifecycle"`
}

type StatisticsFlow struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"userId"`
	Flow        int64  `json:"flow"`
	TotalFlow   int64  `json:"totalFlow"`
	Time        string `json:"time"`
	CreatedTime int64  `json:"createdTime"`
}

type ViteConfig struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Time  int64  `json:"time"`
}

type OutboxItem struct {
	ID          int64           `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	RetryCount  int64           `json:"retryCount"`
	NextRetryAt *int64          `json:"nextRetryAt"`
	CreatedAt   int64           `json:"createdAt"`
	UpdatedAt   int64           `json:"updatedAt"`
}
