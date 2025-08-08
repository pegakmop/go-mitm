package proxy

import (
	"io"
	"net/http"
)

func (p *Proxy) doReplace1(w http.ResponseWriter, r *http.Request) {
	tr := HttpTransport()
	response, err := tr.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	defer func() {
		_ = response.Body.Close()
	}()

	w.WriteHeader(response.StatusCode)
	copyHeader(w.Header(), response.Header)
	_, _ = io.Copy(w, response.Body)
}

func (p *Proxy) doReplace2(w http.ResponseWriter, body []byte) {
	w.WriteHeader(200)
	_, _ = w.Write(body)
}
