package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"sync/atomic"
	"time"
)

func (p *Proxy) getCertificate(domain string) (*tls.Certificate, error) {
	if cert, ok := p.certCache.Load(domain); ok {
		return cert.(*tls.Certificate), nil
	}

	atomic.AddInt64(&p.serialNumber, 1)

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(p.serialNumber),
		Subject: pkix.Name{
			CommonName: domain,
		},
		NotBefore: time.Now().AddDate(0, 0, -1),
		NotAfter:  time.Now().AddDate(1, 0, 0),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	if ip := net.ParseIP(domain); ip != nil {
		serverTemplate.IPAddresses = []net.IP{ip}
	} else {
		serverTemplate.DNSNames = []string{domain}
	}

	// ⚠️ 每个证书都应有自己独立的密钥对
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, serverTemplate, p.rootCert, &serverKey.PublicKey, p.rootKey)
	if err != nil {
		return nil, err
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  serverKey,
	}

	p.certCache.Store(domain, cert)

	return cert, nil
}

var cipherSuiteMap = map[uint16]string{
	0x0005: "TLS_RSA_WITH_RC4_128_SHA",
	0x000a: "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	0x002f: "TLS_RSA_WITH_AES_128_CBC_SHA",
	0x0035: "TLS_RSA_WITH_AES_256_CBC_SHA",
	0x003c: "TLS_RSA_WITH_AES_128_CBC_SHA256",
	0x009c: "TLS_RSA_WITH_AES_128_GCM_SHA256",
	0x009d: "TLS_RSA_WITH_AES_256_GCM_SHA384",
	0xc007: "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
	0xc009: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	0xc00a: "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	0xc011: "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
	0xc012: "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
	0xc013: "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	0xc014: "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	0xc023: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
	0xc027: "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	0xc02f: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	0xc02b: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	0xc030: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	0xc02c: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	0xcca8: "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
	0xcca9: "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
	0x1301: "TLS_AES_128_GCM_SHA256",
	0x1302: "TLS_AES_256_GCM_SHA384",
	0x1303: "TLS_CHACHA20_POLY1305_SHA256",
}

func getReqTLSInfo(reqTLS *tls.ConnectionState) map[string]string {
	tlsInfo := make(map[string]string)
	if reqTLS != nil {
		tlsInfo["ServerName"] = reqTLS.ServerName
		tlsInfo["NegotiatedProtocol"] = reqTLS.NegotiatedProtocol
		tlsInfo["Version"] = fmt.Sprintf("%d", reqTLS.Version)
		tlsInfo["Unique"] = string(reqTLS.TLSUnique)
		tlsInfo["CipherSuite"] = cipherSuiteMap[reqTLS.CipherSuite]
	}
	return tlsInfo
}

// 新增：获取 TLS 信息
func getRespTLSInfo(respTLS, reqTLS *tls.ConnectionState) map[string]string {
	tlsInfo := make(map[string]string)
	if respTLS != nil {
		tlsInfo["ServerName"] = respTLS.ServerName
		tlsInfo["NegotiatedProtocol"] = respTLS.NegotiatedProtocol
		version := "Unknown"
		switch respTLS.Version {
		case tls.VersionTLS10:
			version = "1.0"
		case tls.VersionTLS11:
			version = "1.1"
		case tls.VersionTLS12:
			version = "1.2"
		case tls.VersionTLS13:
			version = "1.3"
		}
		tlsInfo["Version"] = version
		tlsInfo["Unique"] = base64.StdEncoding.EncodeToString(respTLS.TLSUnique)
		if reqTLS != nil {
			if cipherSuite, ok := cipherSuiteMap[reqTLS.CipherSuite]; ok {
				tlsInfo["CipherSuite"] = cipherSuite
			}
		}
	}
	return tlsInfo
}
