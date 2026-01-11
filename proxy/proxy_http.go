package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"
)

func (p *Proxy) handleHttp(rw http.ResponseWriter, r *http.Request) {

	var (
		spend uint16
		size  int64
	)

	reqHeader := make(map[string]string)

	for k := range r.Header {
		reqHeader[k] = r.Header.Get(k)
	}

	reqCookie := make(map[string]string)
	for _, v := range r.Cookies() {
		reqCookie[v.Name] = v.Value
	}

	reqBody := new(bytes.Buffer)

	r.Body = io.NopCloser(io.TeeReader(r.Body, reqBody))

	// 识别 gzip
	acceptEncoding := strings.Split(r.Header.Get("Accept-Encoding"), ",")
	_, bGZIP := dealWithGZIP(acceptEncoding)

	if p.disableGZIP {
		bGZIP = false
	}

	// 获取 代理资源
	reverseProxy := reverseProxyPool.Get().(*httputil.ReverseProxy)
	defer func() {
		reverseProxy.Rewrite = nil
		reverseProxy.Director = nil
		reverseProxy.ModifyResponse = nil
		reverseProxyPool.Put(reverseProxy)
	}()

	var begin time.Time

	// reverseProxy.Director = func(r *http.Request) {
	// 	begin = time.Now()
	// 	r.Header.Set("HOST", r.Host)
	// 	r.Header.Del("Accept-Encoding")
	// 	r.Header.Set("Proxy-Connection", "close")
	// 	// var cookies []string
	// 	// for _, v := range r.Cookies() {
	// 	// 	cookies = append(cookies, v.Name+"="+v.Value)
	// 	// }
	// 	// sort.Strings(cookies)
	// 	// r.Header.Set("cookie", strings.Join(cookies, "; "))
	// }

	reverseProxy.Rewrite = func(req *httputil.ProxyRequest) {
		begin = time.Now()
		req.Out.Header.Set("HOST", r.Host)
		req.Out.Header.Del("Accept-Encoding")
		req.Out.Header.Set("Connection", "close")
	}

	// tr := http.DefaultTransport.(*http.Transport)
	// if p.proxy != nil {
	// 	tr.Proxy = func(_ *http.Request) (*url.URL, error) {
	// 		return p.proxy, nil
	// 	}
	// }
	// tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// reverseProxy.Transport = tr
	reverseProxy.Transport = HttpTransport()

	reverseProxy.ErrorHandler = func(resp http.ResponseWriter, req *http.Request, err error) {
		errFunc("ErrorHandler url(%v) error(%v)\n", req.URL.String(), err)
	}

	reverseProxy.ModifyResponse = func(resp *http.Response) error {
		spend = uint16(time.Since(begin).Milliseconds())

		// 处理响应头
		respHeader := make(map[string]string)
		for k := range resp.Header {
			respHeader[k] = resp.Header.Get(k)
		}

		respCookie := make(map[string]string)
		for _, v := range resp.Cookies() {
			respCookie[v.Name] = v.Value
		}

		respBody := new(bytes.Buffer)

		resp.Body = io.NopCloser(io.TeeReader(resp.Body, respBody))

		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			errFunc("io.ReadAll(resp.Body) error(%v)\n", err)
			return err
		}

		// 拦截修改数据
		if hookFunc != nil {
			responseData = hookFunc(r.URL, responseData)
		}

		size = int64(len(responseData))

		if len(responseData) > 0 {
			// gzip 补充
			if bGZIP {
				resp.Header.Set("Content-Encoding", "gzip")
				responseData = withGZIP(responseData)
			}
			// 重新计算 Content-Length
			resp.Header.Set("Content-Length", strconv.Itoa(len(responseData)))
		}

		// 重写 body
		resp.Body = io.NopCloser(bytes.NewBuffer(responseData))
		if p.messageChan == nil {
			return nil
		}
		p.messageChan <- &Message{
			typ: MessageTypeHTTP,
			data: &HTTPMessage{
				Url:        r.URL.String(),
				RemoteAddr: r.RemoteAddr,
				Method:     r.Method,
				Type:       getContentType(resp.Header),
				Time:       spend,
				Size:       uint16(size),
				Status:     uint16(resp.StatusCode),
				ReqHeader:  reqHeader,
				ReqCookie:  reqCookie,
				ReqBody:    reqBody.String(),
				ReqTls:     getReqTLSInfo(r.TLS),
				RespHeader: respHeader,
				RespCookie: respCookie,
				RespBody:   respBody.String(),
				RespTls:    getRespTLSInfo(resp.TLS, r.TLS),
			},
		}
		return nil
	}

	reverseProxy.ServeHTTP(rw, r)
}
