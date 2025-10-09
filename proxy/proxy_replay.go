package proxy

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
)

func (p *Proxy) Replay(message HTTPMessage) {

	r, err := http.NewRequest(message.Method, message.Url, strings.NewReader(message.ReqBody))
	if err != nil {
		return
	}
	for k, v := range message.ReqHeader {
		r.Header.Set(k, v)
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(reqBody))

	begin := time.Now()
	tr := HttpTransport()
	response, err := tr.RoundTrip(r)
	spend := uint16(time.Since(begin).Milliseconds())
	if err != nil {
		return
	}

	defer func() {
		_ = response.Body.Close()
	}()

	var size int64
	var respBody string
	contentTypes := response.Header.Get("Content-Type")
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	size = int64(len(bodyBytes))
	if !(strings.Contains(strings.ToLower(contentTypes), "image") || strings.Contains(strings.ToLower(contentTypes), "video")) {
		if response.Header.Get("Content-Encoding") == "deflate" {
			reader := flate.NewReader(bytes.NewReader(bodyBytes))
			defer func() {
				err = reader.Close()
				if err != nil {
					return
				}
			}()

			bodyBytes, err = io.ReadAll(reader)
			if err != nil {
				return
			}
		}
		if response.Header.Get("Content-Encoding") == "br" {
			bodyBytes, err = io.ReadAll(brotli.NewReader(bytes.NewReader(bodyBytes)))
			if err != nil {
				return
			}
		}
		if response.Header.Get("Content-Encoding") == "deflate" {
			reader := flate.NewReader(bytes.NewReader(bodyBytes))
			defer func() {
				if reader != nil {
					err = reader.Close()
					if err != nil {
						return
					}
				}
			}()
			bodyBytes, err = io.ReadAll(reader)
			if err != nil {
				return
			}
		}
		if response.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(bytes.NewReader(bodyBytes))
			defer func() {
				if reader != nil {
					err = reader.Close()
					if err != nil {
						return
					}
				}
			}()
			if err != nil {
				return
			}
			bodyBytes, err = io.ReadAll(reader)
			if err != nil {
				return
			}
		}

		respBody = string(bodyBytes)
	}

	go func(r *http.Request, response *http.Response) {
		reqHeader := make(map[string]string)
		for k := range r.Header {
			reqHeader[k] = r.Header.Get(k)
		}
		respHeader := make(map[string]string)
		for k := range response.Header {
			respHeader[k] = response.Header.Get(k)
		}

		//reqTrailer := make(map[string]string)
		//for k := range r.Trailer {
		//	reqTrailer[k] = r.Trailer.Get(k)
		//}
		//respTrailer := make(map[string]string)
		//for k := range response.Trailer {
		//	respTrailer[k] = response.Trailer.Get(k)
		//}

		reqCookie := make(map[string]string)
		for _, v := range r.Cookies() {
			reqCookie[v.Name] = v.Raw
		}
		respCookie := make(map[string]string)
		for _, v := range response.Cookies() {
			respCookie[v.Name] = v.Raw
		}

		contentType := contentTypes
		for _, v := range strings.Split(contentTypes, ";") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if strings.Contains(strings.ToLower(v), "charset=") {
				continue
			}
			contentType = v
			break
		}

		//p.logger.Info("Response", "StatusCode", response.StatusCode, r.Method, r.URL.String(), "contentType", contentType)

		if p.messageChan != nil {

			p.messageChan <- &Message{
				typ: MessageTypeHTTP,
				data: &HTTPMessage{
					Url:        r.URL.String(),
					RemoteAddr: r.RemoteAddr,
					Method:     r.Method,
					Type:       contentType,
					Time:       spend,
					Size:       uint16(size),
					Status:     uint16(response.StatusCode),
					ReqHeader:  reqHeader,
					ReqCookie:  reqCookie,
					ReqBody:    string(reqBody),
					ReqTls:     getReqTLSInfo(r.TLS),
					RespHeader: respHeader,
					RespCookie: respCookie,
					RespBody:   respBody,
					RespTls:    getRespTLSInfo(response.TLS, r.TLS),
				},
			}

		}
	}(r, response)
}
