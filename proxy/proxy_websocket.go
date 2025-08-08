package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// 新增：处理 WebSocket 连接
func (p *Proxy) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 复制原始请求头
	header := make(http.Header)
	for k, v := range r.Header {
		switch k {
		case "Upgrade":
		case "Connection":
		case "Sec-Websocket-Key":
		case "Sec-Websocket-Version":
		case "Sec-Websocket-Extensions":
		default:
			header[k] = v
		}
	}

	// 创建到目标服务器的 WebSocket 连接
	targetURL := r.URL

	if targetURL.Scheme == "" {
		u, _ := url.Parse(r.Header.Get("origin"))
		targetURL.Scheme = u.Scheme
	}

	if targetURL.Host == "" {
		targetURL.Host = r.Host
	}

	switch targetURL.Scheme {
	case "http":
		targetURL.Scheme = "ws"
	case "https":
		targetURL.Scheme = "wss"
	}

	uri := fmt.Sprintf("%s://%s%s", targetURL.Scheme, targetURL.Host, targetURL.Path)

	if querys := targetURL.Query(); len(querys) > 0 {
		uri += "?" + querys.Encode()
	}

	// 连接目标 WebSocket 服务器
	dialer := WebSocketDialer().Dialer()

	targetConn, resp, err := dialer.Dial(uri, header)
	if err != nil {
		if resp != nil {
			copyHeader(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	defer targetConn.Close()

	upgradeHeader := http.Header{}
	upgradeHeader.Add("Sec-WebSocket-Protocol", r.Header.Get("Sec-WebSocket-Protocol"))

	for _, v := range dialer.Jar.Cookies(r.URL) {
		upgradeHeader.Add("Set-Cookie", v.String())
	}

	// 升级客户端连接为 WebSocket
	clientConn, err := wsUpgrader.Upgrade(w, r, upgradeHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// 记录 WebSocket 连接信息
	var msg *Message
	if p.messageChan != nil {
		msg = &Message{
			Url:        r.URL.String(),
			RemoteAddr: r.RemoteAddr,
			Method:     r.Method,
			Type:       "websocket",
			Status:     101,
			ReqHeader: map[string]string{
				"Upgrade":    r.Header.Get("Upgrade"),
				"Connection": r.Header.Get("Connection"),
			},
			RespBodyChan: make(chan []byte, 10240),
		}
		defer close(msg.RespBodyChan)

		p.messageChan <- msg
	}

	var g sync.WaitGroup
	g.Add(2)
	go func() {
		defer g.Done()
		p.proxyWebSocket(clientConn, targetConn, msg)
	}()

	go func() {
		defer g.Done()
		p.proxyWebSocket(targetConn, clientConn, nil)
	}()
	g.Wait()
}

// 新增：转发 WebSocket 消息
func (p *Proxy) proxyWebSocket(dst, src *websocket.Conn, msg *Message) error {

	for {
		messageType, message, err := src.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				return nil
			}
			return err
		}

		err = dst.WriteMessage(messageType, message)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				return nil
			}
			return err
		}

		if msg != nil {
			msg.RespBodyChan <- message
		}
	}
}
