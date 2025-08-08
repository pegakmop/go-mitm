package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	utls "github.com/refraction-networking/utls"
)

var (
	clientHelloConfig *UtlsConfig

	clientHelloMap = map[string]utls.ClientHelloID{
		utls.HelloGolang.Str(): utls.HelloGolang,

		utls.HelloCustom.Str(): utls.HelloCustom,

		utls.HelloRandomized.Str():       utls.HelloRandomized,
		utls.HelloRandomizedALPN.Str():   utls.HelloRandomizedALPN,
		utls.HelloRandomizedNoALPN.Str(): utls.HelloRandomizedNoALPN,

		"firefox":                   utls.HelloFirefox_Auto,
		utls.HelloFirefox_55.Str():  utls.HelloFirefox_55,
		utls.HelloFirefox_56.Str():  utls.HelloFirefox_56,
		utls.HelloFirefox_63.Str():  utls.HelloFirefox_63,
		utls.HelloFirefox_65.Str():  utls.HelloFirefox_65,
		utls.HelloFirefox_99.Str():  utls.HelloFirefox_99,
		utls.HelloFirefox_102.Str(): utls.HelloFirefox_102,
		utls.HelloFirefox_105.Str(): utls.HelloFirefox_105,
		utls.HelloFirefox_120.Str(): utls.HelloFirefox_120,

		"chrome":                                    utls.HelloChrome_Auto,
		utls.HelloChrome_58.Str():                   utls.HelloChrome_58,
		utls.HelloChrome_62.Str():                   utls.HelloChrome_62,
		utls.HelloChrome_70.Str():                   utls.HelloChrome_70,
		utls.HelloChrome_72.Str():                   utls.HelloChrome_72,
		utls.HelloChrome_83.Str():                   utls.HelloChrome_83,
		utls.HelloChrome_87.Str():                   utls.HelloChrome_87,
		utls.HelloChrome_96.Str():                   utls.HelloChrome_96,
		utls.HelloChrome_100.Str():                  utls.HelloChrome_100,
		utls.HelloChrome_102.Str():                  utls.HelloChrome_102,
		utls.HelloChrome_106_Shuffle.Str():          utls.HelloChrome_106_Shuffle,
		utls.HelloChrome_100_PSK.Str():              utls.HelloChrome_100_PSK,
		utls.HelloChrome_112_PSK_Shuf.Str():         utls.HelloChrome_112_PSK_Shuf,
		utls.HelloChrome_114_Padding_PSK_Shuf.Str(): utls.HelloChrome_114_Padding_PSK_Shuf,
		utls.HelloChrome_115_PQ.Str():               utls.HelloChrome_115_PQ,
		utls.HelloChrome_115_PQ_PSK.Str():           utls.HelloChrome_115_PQ_PSK,
		utls.HelloChrome_120.Str():                  utls.HelloChrome_120,
		utls.HelloChrome_120_PQ.Str():               utls.HelloChrome_120_PQ,
		utls.HelloChrome_131.Str():                  utls.HelloChrome_131,

		"ios":                    utls.HelloIOS_Auto,
		utls.HelloIOS_11_1.Str(): utls.HelloIOS_11_1,
		utls.HelloIOS_12_1.Str(): utls.HelloIOS_12_1,
		utls.HelloIOS_13.Str():   utls.HelloIOS_13,
		utls.HelloIOS_14.Str():   utls.HelloIOS_14,

		"android":                         utls.HelloAndroid_11_OkHttp,
		utls.HelloAndroid_11_OkHttp.Str(): utls.HelloAndroid_11_OkHttp,

		"edge":                   utls.HelloEdge_Auto,
		utls.HelloEdge_85.Str():  utls.HelloEdge_85,
		utls.HelloEdge_106.Str(): utls.HelloEdge_106,

		"safari":                    utls.HelloSafari_Auto,
		utls.HelloSafari_16_0.Str(): utls.HelloSafari_16_0,

		"360":                    utls.Hello360_Auto,
		utls.Hello360_7_5.Str():  utls.Hello360_7_5,
		utls.Hello360_11_0.Str(): utls.Hello360_11_0,

		"QQ":                    utls.HelloQQ_Auto,
		utls.HelloQQ_11_1.Str(): utls.HelloQQ_11_1,
	}
)

type UtlsConfig struct {
	Fingerprint string // default utls.HelloRandomizedNoALPN
	ALPN        string // default "http/1.1"
}

func Client(conn net.Conn, addr string) (net.Conn, error) {
	serverName, _, _ := net.SplitHostPort(addr)
	return tls.Client(conn, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true,
	}), nil
}

func UClient(ctx context.Context, conn net.Conn, addr string, conf *UtlsConfig) (net.Conn, error) {
	serverName, _, _ := net.SplitHostPort(addr)

	alpn := conf.ALPN
	if alpn == "" {
		alpn = "http/1.1"
	}

	clientHelloID, ok := clientHelloMap[conf.Fingerprint]
	if !ok {
		clientHelloID = utls.HelloRandomizedNoALPN
	}

	uTlsConn := utls.UClient(conn, &utls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true,
	}, clientHelloID)

	spec, err := utls.UTLSIdToSpec(clientHelloID)
	if err != nil {
		return nil, err
	}

	bAddALPNExtension := true
	for _, v := range spec.Extensions {
		if _, ok := v.(*utls.ALPNExtension); !ok {
			continue
		}
		bAddALPNExtension = false
		break
	}

	if bAddALPNExtension {
		spec.Extensions = append(spec.Extensions, &utls.ALPNExtension{AlpnProtocols: []string{alpn}})
	}

	if err := uTlsConn.ApplyPreset(&spec); err != nil {
		return nil, fmt.Errorf("ApplyPreset failed: %w", err)
	}

	if err := uTlsConn.HandshakeContext(ctx); err != nil {
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}
	return uTlsConn, nil
}
