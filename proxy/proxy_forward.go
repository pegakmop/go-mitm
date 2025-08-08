package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

func (p *Proxy) forward(w http.ResponseWriter, r *http.Request) {
	r.Header.Del("Proxy-Connection")

	if r.Method == "CONNECT" {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()

		conn, err := new(net.Dialer).DialContext(ctx, "tcp", r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		_, _ = w.Write([]byte("HTTP/1.1 200 Connection Established\n\n"))
		// if _, err = fmt.Fprint(w, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
		// 	http.Error(w, err.Error(), http.StatusServiceUnavailable)
		// 	return
		// }

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
			return
		}

		hijack, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
		defer func() {
			_ = hijack.Close()
		}()

		var g sync.WaitGroup
		g.Add(2)
		go func() {
			defer g.Done()
			io.Copy(conn, hijack)
		}()
		go func() {
			defer g.Done()
			io.Copy(hijack, conn)
		}()
		g.Wait()
	} else {
		reverseProxy := reverseProxyPool.Get().(*httputil.ReverseProxy)
		defer func() {
			reverseProxy.Rewrite = nil
			reverseProxy.ModifyResponse = nil
			reverseProxyPool.Put(reverseProxy)
		}()

		reverseProxy.Rewrite = func(req *httputil.ProxyRequest) {
			// req.Out.Header.Set("HOST", r.Host)
			// req.Out.Header.Del("Accept-Encoding")
		}

		reverseProxy.Transport = HttpTransport()
		reverseProxy.ServeHTTP(w, r)
	}
}
