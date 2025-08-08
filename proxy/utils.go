package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

func LanIp() (lanIp string) {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			addr, err := iface.Addrs()
			if err != nil {
				fmt.Println(err)
				continue
			}

			for _, addr := range addr {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
					lanIp = ipNet.IP.String()
					break
				}
			}
		}
	}
	return
}

func InternetIp() (internetIp string) {
	resp, err := http.Get("https://api64.ipify.org?format=text")
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if err == nil {
		_, _ = fmt.Fscanf(resp.Body, "%s", &internetIp)
	}
	return
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func isSSERequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	contentType := r.Header.Get("Content-Type")
	return strings.Contains(strings.ToLower(contentType), "text/event-stream") ||
		strings.Contains(strings.ToLower(contentType), "application/x-ndjson") ||
		strings.Contains(strings.ToLower(contentType), "multipart/x-mixed-replace") ||
		strings.Contains(strings.ToLower(accept), "text/event-stream")
}

// 新增：检查是否为 WebSocket 请求
func isWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// 新增：提取 Content-Type
func getContentType(header http.Header) string {
	contentType := header.Get("Content-Type")
	for _, v := range strings.Split(contentType, ";") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if strings.Contains(strings.ToLower(v), "charset=") {
			continue
		}
		return v
	}
	return ""
}

// 从 RemoteAddr 中提取 IP 地址
func getRealIP(req *http.Request) string {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return ""
	}
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
