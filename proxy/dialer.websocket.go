package proxy

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/gorilla/websocket"
)

var (
	dialer = &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		// EnableCompression: true,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Jar:             cookieJar(),
	}
	// 添加 WebSocket upgrader
	wsUpgrader = websocket.Upgrader{
		// EnableCompression: true,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func cookieJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
}

type WebSocket websocket.Dialer

func (ws *WebSocket) Dialer() *websocket.Dialer {
	return (*websocket.Dialer)(ws)
}

func (ws *WebSocket) Clone() *WebSocket {
	return &WebSocket{
		Proxy:             ws.Proxy,
		HandshakeTimeout:  ws.HandshakeTimeout,
		EnableCompression: ws.EnableCompression,
		TLSClientConfig:   ws.TLSClientConfig,
		NetDialContext:    ws.NetDialContext,
		NetDial:           ws.NetDial,
		NetDialTLSContext: ws.NetDialTLSContext,
		ReadBufferSize:    ws.ReadBufferSize,
		WriteBufferSize:   ws.WriteBufferSize,
		WriteBufferPool:   ws.WriteBufferPool,
		Subprotocols:      ws.Subprotocols,
		Jar:               ws.Jar,
	}
}

func WebSocketDialer() *WebSocket {
	dialer.NetDialContext = netDialer.DialContext
	dialer.NetDialTLSContext = netDialer.DialTLSContext

	if proxyFunc != nil && dialer.Proxy == nil {
		dialer.Proxy = proxyFunc
	}

	if proxyFunc == nil && dialer.Proxy != nil {
		dialer.Proxy = nil
	}
	return (*WebSocket)(dialer)
}
