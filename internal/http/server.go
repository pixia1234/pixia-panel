package httpapi

import (
	"net/http"
	"time"

	"pixia-panel/internal/captcha"
	"pixia-panel/internal/flow"
	"pixia-panel/internal/gost"
	"pixia-panel/internal/store"
)

type Server struct {
	store     *store.Store
	flow      *flow.Service
	hub       *gost.Hub
	captcha   *captcha.Service
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewServer(store *store.Store, flow *flow.Service, hub *gost.Hub, captcha *captcha.Service, jwtSecret []byte, tokenTTL time.Duration) *Server {
	return &Server{store: store, flow: flow, hub: hub, captcha: captcha, jwtSecret: jwtSecret, tokenTTL: tokenTTL}
}

func (s *Server) Register(mux *http.ServeMux) {
	// public endpoints
	mux.HandleFunc("/flow/test", s.handleFlowTest)
	mux.HandleFunc("/flow/upload", s.handleFlowUpload)
	mux.HandleFunc("/flow/config", s.handleFlowConfig)
	mux.HandleFunc("/api/v1/captcha/check", s.handleCaptchaCheck)
	mux.HandleFunc("/api/v1/captcha/generate", s.handleCaptchaGenerate)
	mux.HandleFunc("/api/v1/captcha/verify", s.handleCaptchaVerify)
	mux.HandleFunc("/api/v1/open_api/sub_store", s.handleOpenAPISubStore)
	mux.HandleFunc("/api/v1/config/list", s.handleConfigList)
	mux.HandleFunc("/api/v1/config/get", s.handleConfigGet)

	// auth endpoints
	mux.HandleFunc("/api/v1/user/login", s.handleUserLogin)

	// protected endpoints
	protected := func(path string, h http.Handler) {
		mux.Handle(path, withAuth(s.jwtSecret, h))
	}

	admin := func(path string, h http.Handler) {
		mux.Handle(path, withAuth(s.jwtSecret, requireAdmin(h)))
	}

	protected("/api/v1/user/package", http.HandlerFunc(s.handleUserPackage))
	protected("/api/v1/user/updatePassword", http.HandlerFunc(s.handleUserUpdatePassword))
	admin("/api/v1/user/create", http.HandlerFunc(s.handleUserCreate))
	admin("/api/v1/user/list", http.HandlerFunc(s.handleUserList))
	admin("/api/v1/user/update", http.HandlerFunc(s.handleUserUpdate))
	admin("/api/v1/user/delete", http.HandlerFunc(s.handleUserDelete))
	admin("/api/v1/user/reset", http.HandlerFunc(s.handleUserResetFlow))

	admin("/api/v1/node/create", http.HandlerFunc(s.handleNodeCreate))
	admin("/api/v1/node/list", http.HandlerFunc(s.handleNodeList))
	admin("/api/v1/node/update", http.HandlerFunc(s.handleNodeUpdate))
	admin("/api/v1/node/delete", http.HandlerFunc(s.handleNodeDelete))
	admin("/api/v1/node/install", http.HandlerFunc(s.handleNodeInstall))
	admin("/api/v1/node/check-status", http.HandlerFunc(s.handleNodeCheckStatus))

	admin("/api/v1/tunnel/create", http.HandlerFunc(s.handleTunnelCreate))
	admin("/api/v1/tunnel/list", http.HandlerFunc(s.handleTunnelList))
	admin("/api/v1/tunnel/get", http.HandlerFunc(s.handleTunnelGet))
	admin("/api/v1/tunnel/update", http.HandlerFunc(s.handleTunnelUpdate))
	admin("/api/v1/tunnel/delete", http.HandlerFunc(s.handleTunnelDelete))
	admin("/api/v1/tunnel/user/assign", http.HandlerFunc(s.handleUserTunnelAssign))
	admin("/api/v1/tunnel/user/list", http.HandlerFunc(s.handleUserTunnelList))
	admin("/api/v1/tunnel/user/remove", http.HandlerFunc(s.handleUserTunnelRemove))
	admin("/api/v1/tunnel/user/update", http.HandlerFunc(s.handleUserTunnelUpdate))
	protected("/api/v1/tunnel/user/tunnel", http.HandlerFunc(s.handleUserTunnelAvailable))
	admin("/api/v1/tunnel/diagnose", http.HandlerFunc(s.handleTunnelDiagnose))

	protected("/api/v1/forward/create", http.HandlerFunc(s.handleForwardCreate))
	protected("/api/v1/forward/list", http.HandlerFunc(s.handleForwardList))
	protected("/api/v1/forward/update", http.HandlerFunc(s.handleForwardUpdate))
	protected("/api/v1/forward/delete", http.HandlerFunc(s.handleForwardDelete))
	protected("/api/v1/forward/force-delete", http.HandlerFunc(s.handleForwardForceDelete))
	protected("/api/v1/forward/pause", http.HandlerFunc(s.handleForwardPause))
	protected("/api/v1/forward/resume", http.HandlerFunc(s.handleForwardResume))
	protected("/api/v1/forward/diagnose", http.HandlerFunc(s.handleForwardDiagnose))
	protected("/api/v1/forward/update-order", http.HandlerFunc(s.handleForwardUpdateOrder))

	admin("/api/v1/speed-limit/create", http.HandlerFunc(s.handleSpeedLimitCreate))
	admin("/api/v1/speed-limit/list", http.HandlerFunc(s.handleSpeedLimitList))
	admin("/api/v1/speed-limit/update", http.HandlerFunc(s.handleSpeedLimitUpdate))
	admin("/api/v1/speed-limit/delete", http.HandlerFunc(s.handleSpeedLimitDelete))
	admin("/api/v1/speed-limit/tunnels", http.HandlerFunc(s.handleSpeedLimitTunnels))

	admin("/api/v1/config/update", http.HandlerFunc(s.handleConfigUpdateBatch))
	admin("/api/v1/config/update-single", http.HandlerFunc(s.handleConfigUpdateSingle))
}
