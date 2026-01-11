package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	proxyFunc func(*http.Request) (*url.URL, error)
	hookFunc  func(*url.URL, []byte) []byte
	errorFunc func(format string, a ...any)

	netDialer Dialer = NewNetDialer(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	})
)

type Proxy struct {
	wg           sync.WaitGroup
	rootCert     *x509.Certificate
	rootKey      *rsa.PrivateKey
	privateKey   *rsa.PrivateKey
	listener     *Listener
	socks5       string
	disableGZIP  bool
	httpSrv      *http.Server
	httpsSrv     *http.Server
	serialNumber int64
	messageChan  chan *Message
	exclude      []string
	include      []string
	replace      [][]string
	logger       *slog.Logger
	certCache    sync.Map
}

func (p *Proxy) SetMessageChan(messageChan chan *Message) {
	p.messageChan = messageChan
}

func (p *Proxy) Include() []string {
	return p.include
}

func (p *Proxy) SetInclude(includes string) []string {
	include := make([]string, 0)
	for _, v := range strings.Split(includes, ";") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		include = append(include, v)
	}
	p.include = include
	return p.include
}

func (p *Proxy) ClearInclude() []string {
	p.include = make([]string, 0)
	return p.include
}

func (p *Proxy) Exclude() []string {
	return p.exclude
}

func (p *Proxy) SetExclude(excludes string) []string {
	exclude := make([]string, 0)
	for _, v := range strings.Split(excludes, ";") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		exclude = append(exclude, v)
	}
	p.exclude = exclude
	return p.exclude
}

func (p *Proxy) ClearExclude() []string {
	p.exclude = make([]string, 0)
	return p.exclude
}

func (p *Proxy) Replace() [][]string {
	return p.replace
}

func (p *Proxy) SetReplace(replaces string) [][]string {
	replace := make([][]string, 0)
	for _, v := range strings.Split(replaces, ";") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		replace = append(replace, strings.Split(v, ","))
	}
	p.replace = replace
	return p.replace
}

func (p *Proxy) ClearReplace() [][]string {
	p.replace = make([][]string, 0)
	return p.replace
}

func SetProxy(uri string) {
	if uri == "" {
		return
	}
	proxyFunc = func(_ *http.Request) (*url.URL, error) {
		return url.Parse(uri)
	}
}

func ClearProxy() {
	proxyFunc = nil
}

func (p *Proxy) Socks5() string {
	return p.socks5
}

func (p *Proxy) SetSocks5(addr string) {
	if addr == "" {
		return
	}

	p.socks5 = addr

	netDialer, _ = NewSocks5Dialer(addr)
}

func (p *Proxy) ClearSocks5() {
	p.socks5 = ""
	netDialer = NewNetDialer(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	})
}

func ClientHello() *UtlsConfig {
	return clientHelloConfig
}

func SetClientHello(config *UtlsConfig) {
	clientHelloConfig = config
}

func ClearClientHello() {
	clientHelloConfig = nil
}

// hookFunc   func(*url.URL, []byte) []byte
func SetHook(hook func(*url.URL, []byte) []byte) {
	hookFunc = hook
}

func ClearHook() {
	hookFunc = nil
}

func (p *Proxy) DisableGZIP() {
	p.disableGZIP = true
}

func SetError(errFn func(format string, a ...any)) {
	errorFunc = errFn
}

func ClearError() {
	errorFunc = nil
}

func NewProxy(addr string, caCert, caKey []byte) (p *Proxy, err error) {
	p = new(Proxy)
	p.logger = slog.Default()

	{ // ca.cert
		block, _ := pem.Decode(caCert)
		if block == nil {
			return
		}

		if p.rootCert, err = x509.ParseCertificate(block.Bytes); err != nil {
			return
		}
	}

	{ // ca.key
		block, _ := pem.Decode(caKey)
		if block == nil {
			return
		}

		if p.rootKey, err = x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
			return
		}
	}

	// server.key
	if p.privateKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
		return
	}

	p.listener, _ = NewListener()

	p.httpsSrv = &http.Server{
		Handler: p,
		TLSConfig: &tls.Config{
			GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return p.getCertificate(info.ServerName)
			},
		},
	}

	p.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      p,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	return
}

func (p *Proxy) Start() {
	p.wg.Add(2)
	go func() {
		defer p.wg.Done()
		p.httpsSrv.ServeTLS(p.listener, "", "")
	}()

	go func() {
		defer p.wg.Done()
		p.httpSrv.ListenAndServe()
	}()

}

func (p *Proxy) Stop() {
	p.httpsSrv.Close()
	p.httpSrv.Close()

	p.wg.Wait()
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if strings.Contains(r.Host, ":") {
		host = host[:strings.Index(host, ":")]
	}

	for _, v := range p.exclude {
		if matched, _ := filepath.Match(v, host); matched {
			p.forward(w, r)
			return
		}
	}

	include := true
	for _, v := range p.include {
		include = false
		if matched, _ := filepath.Match(v, host); matched {
			include = true
			break
		}
	}

	if !include {
		p.forward(w, r)
		return
	}

	if isHttpsRequest(r) {
		p.handleHttps(w, r)
		return
	}
	// 检查是否为 WebSocket 请求
	if isWebSocketRequest(r) {
		p.handleWebSocket(w, r)
		return
	}

	if isSSERequest(r) {
		p.handleSSE(w, r)
		return
	}

	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}

	if r.URL.Scheme == "" {
		r.URL.Scheme = "https"
	}

	for _, v := range p.replace {
		if ok, _ := filepath.Match(v[0], r.URL.String()); !ok {
			continue
		}

		switch v[1] {
		case "http://", "https://":
			if r, err := http.NewRequest(http.MethodGet, v[2], nil); err == nil {
				p.doReplace1(w, r)
			}
		case "file://":
			if data, err := os.ReadFile("/" + v[2]); err == nil {
				p.doReplace2(w, data)
			}
		}
		return
	}
	p.handleHttp(w, r)
}
