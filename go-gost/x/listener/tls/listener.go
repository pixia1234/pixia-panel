package tls

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/go-gost/core/limiter"
	"github.com/go-gost/core/listener"
	"github.com/go-gost/core/logger"
	md "github.com/go-gost/core/metadata"
	admission "github.com/go-gost/x/admission/wrapper"
	xnet "github.com/go-gost/x/internal/net"
	"github.com/go-gost/x/internal/net/proxyproto"
	climiter "github.com/go-gost/x/limiter/conn/wrapper"
	limiter_wrapper "github.com/go-gost/x/limiter/traffic/wrapper"
	metrics "github.com/go-gost/x/metrics/wrapper"
	stats "github.com/go-gost/x/observer/stats/wrapper"
	"github.com/go-gost/x/registry"
)

func init() {
	registry.ListenerRegistry().Register("tls", NewListener)
}

type tlsListener struct {
	ln      net.Listener
	logger  logger.Logger
	md      metadata
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &tlsListener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *tlsListener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	network := "tcp"
	if xnet.IsIPv4(l.options.Addr) {
		network = "tcp4"
	}

	lc := net.ListenConfig{}
	if l.md.mptcp {
		lc.SetMultipathTCP(true)
		l.logger.Debugf("mptcp enabled: %v", lc.MultipathTCP())
	}
	ln, err := lc.Listen(context.Background(), network, l.options.Addr)
	if err != nil {
		return
	}
	ln = proxyproto.WrapListener(l.options.ProxyProtocol, ln, 10*time.Second)
	ln = metrics.WrapListener(l.options.Service, ln)
	ln = stats.WrapListener(ln, l.options.Stats)
	ln = admission.WrapListener(l.options.Admission, ln)
	ln = limiter_wrapper.WrapListener(l.options.Service, ln, l.options.TrafficLimiter)
	ln = climiter.WrapListener(l.options.ConnLimiter, ln)

	l.ln = newSilentTLSListener(ln, l.options.TLSConfig, l.logger)

	return
}

func (l *tlsListener) Accept() (conn net.Conn, err error) {
	conn, err = l.ln.Accept()
	if err != nil {
		return
	}

	conn = limiter_wrapper.WrapConn(
		conn,
		l.options.TrafficLimiter,
		conn.RemoteAddr().String(),
		limiter.ScopeOption(limiter.ScopeConn),
		limiter.ServiceOption(l.options.Service),
		limiter.NetworkOption(conn.LocalAddr().Network()),
		limiter.SrcOption(conn.RemoteAddr().String()),
	)

	return
}

func (l *tlsListener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *tlsListener) Close() error {
	return l.ln.Close()
}

// silentTLSListener 是一个自定义的TLS监听器，对非TLS连接静默处理
type silentTLSListener struct {
	net.Listener
	config *tls.Config
	logger logger.Logger
}

func newSilentTLSListener(inner net.Listener, config *tls.Config, logger logger.Logger) net.Listener {
	return &silentTLSListener{
		Listener: inner,
		config:   config,
		logger:   logger,
	}
}

func (l *silentTLSListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		// 创建TLS连接
		tlsConn := tls.Server(conn, l.config)

		// 设置握手超时
		tlsConn.SetDeadline(time.Now().Add(10 * time.Second))

		// 尝试TLS握手
		if err := tlsConn.Handshake(); err != nil {

			conn.Close()
			continue // 继续接受下一个连接
		}

		// 清除超时设置
		tlsConn.SetDeadline(time.Time{})

		return tlsConn, nil
	}
}
