package proxy

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

var (
	tr = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		DisableCompression:    true,
	}

	reverseProxyPool = &sync.Pool{
		New: func() any {
			return &httputil.ReverseProxy{ /*Transport: tr*/ }
		},
	}
)

func HttpTransport() http.RoundTripper {
	tr.DialContext = netDialer.DialContext
	tr.DialTLSContext = netDialer.DialTLSContext

	if proxyFunc != nil && tr.Proxy == nil {
		tr.Proxy = proxyFunc
	}
	if proxyFunc == nil && tr.Proxy != nil {
		tr.Proxy = nil
	}
	return tr

	// spec := &tlsutls.ClientHelloSpec{
	// 	TLSVersMin: tls.VersionTLS12,
	// 	TLSVersMax: tls.VersionTLS13,
	// 	CipherSuites: []uint16{
	// 		tls.TLS_AES_128_GCM_SHA256, // TLS 1.3
	// 		tls.TLS_AES_256_GCM_SHA384, // TLS 1.3
	// 		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	// 		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	// 	},
	// 	Extensions: []tlsutls.TLSExtension{
	// 		&tlsutls.SNIExtension{},
	// 		&tlsutls.SupportedCurvesExtension{Curves: []tlsutls.CurveID{tlsutls.GREASE_PLACEHOLDER, tlsutls.X25519, tlsutls.CurveP256}},
	// 		&tlsutls.SupportedPointsExtension{SupportedPoints: []byte{0}},
	// 		&tlsutls.SignatureAlgorithmsExtension{
	// 			SupportedSignatureAlgorithms: []tlsutls.SignatureScheme{
	// 				tlsutls.ECDSAWithP256AndSHA256,
	// 				tlsutls.PSSWithSHA256,
	// 				tlsutls.PKCS1WithSHA256,
	// 			},
	// 		},
	// 		&tlsutls.SupportedVersionsExtension{
	// 			Versions: []uint16{tls.VersionTLS13, tls.VersionTLS12},
	// 		},
	// 		&tlsutls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
	// 		&tlsutls.StatusRequestExtension{},
	// 		&tlsutls.UtlsExtendedMasterSecretExtension{},
	// 	},
	// }

	// spec, _ := tlsutls.UTLSIdToSpec(tlsutls.HelloRandomized)
	// for i, ext := range spec.Extensions {
	// 	if _, ok := ext.(*tlsutls.ALPNExtension); ok {
	// 		spec.Extensions[i] = &tlsutls.ALPNExtension{AlpnProtocols: []string{"http/1.1"}}
	// 	}
	// }

	// return &UTLSTransport{
	// 	Socks5Addr: "127.0.0.1:10808",
	// 	Spec:       &spec,
	// 	Timeout:    10 * time.Second,
	// }
}
