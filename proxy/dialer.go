package proxy

import (
	"context"
	"net"

	"golang.org/x/net/proxy"
)

type Dialer interface {
	// Dial connects to the given address via the proxy.
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	DialTLSContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type NetDialer struct {
	dialer *net.Dialer
}

func NewNetDialer(dialer *net.Dialer) Dialer {
	return &NetDialer{dialer: dialer}
}

func (d *NetDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.dialer.DialContext(ctx, network, addr)
}

func (d *NetDialer) DialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	if clientHelloConfig == nil {
		return Client(conn, addr)
	}

	return UClient(ctx, conn, addr, clientHelloConfig)
}

type Socks5Dialer struct {
	dialer proxy.Dialer
}

func NewSocks5Dialer(addr string) (Dialer, error) {
	dialer, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return &Socks5Dialer{dialer: dialer}, nil
}

func (d *Socks5Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	ch := make(chan *sockConn, 1)
	defer close(ch)

	go func() {
		conn, err := d.dialer.Dial(network, addr)
		select {
		case <-ctx.Done():
			// 如果超时/取消了，这里不再发送，避免阻塞
		case ch <- &sockConn{conn: conn, err: err}:
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case d := <-ch:
		return d.conn, d.err
	}
}

func (d *Socks5Dialer) DialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {

	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	if clientHelloConfig == nil {
		return Client(conn, addr)
	}
	return UClient(ctx, conn, addr, clientHelloConfig)
}

type sockConn struct {
	conn net.Conn
	err  error
}
