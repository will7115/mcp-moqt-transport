package mcpmoqt

import (
	"context"
	"fmt"
	"net"

	"github.com/mengelbart/moqtransport"
	"github.com/mengelbart/moqtransport/quicmoq"
	"github.com/quic-go/quic-go"
)

type moqConnWithAddr struct {
	moqtransport.Connection
	remoteAddr net.Addr
}

func (c moqConnWithAddr) RemoteAddr() net.Addr { return c.remoteAddr }

func listenAndAcceptMOQT(ctx context.Context, cfg *transportConfig) (moqtransport.Connection, net.Addr, func() error, error) {
	if err := ensureServerTLS(cfg); err != nil {
		return nil, nil, nil, err
	}
	listener, err := quic.ListenAddr(cfg.addr, cfg.tlsServer, cfg.quicConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("quic listen: %w", err)
	}
	closeListener := func() error { return listener.Close() }

	conn, err := listener.Accept(ctx)
	if err != nil {
		_ = listener.Close()
		return nil, nil, nil, fmt.Errorf("quic accept: %w", err)
	}
	// Single-connection transport for MCP Run/Connect: close listener after accept.
	_ = listener.Close()

	if conn.ConnectionState().TLS.NegotiatedProtocol == "" {
		_ = conn.CloseWithError(0, "")
		return nil, nil, nil, fmt.Errorf("quic negotiated empty ALPN")
	}
	if conn.ConnectionState().TLS.NegotiatedProtocol != "moq-00" {
		_ = conn.CloseWithError(0, "")
		return nil, nil, nil, fmt.Errorf("unexpected ALPN: %s", conn.ConnectionState().TLS.NegotiatedProtocol)
	}

	return moqConnWithAddr{Connection: quicmoq.NewServer(conn), remoteAddr: conn.RemoteAddr()}, conn.RemoteAddr(), closeListener, nil
}

func dialMOQT(ctx context.Context, cfg *transportConfig) (moqtransport.Connection, net.Addr, error) {
	if err := ensureClientTLS(cfg); err != nil {
		return nil, nil, err
	}
	conn, err := quic.DialAddr(ctx, cfg.addr, cfg.tlsClient, cfg.quicConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("quic dial: %w", err)
	}
	if conn.ConnectionState().TLS.NegotiatedProtocol == "" {
		_ = conn.CloseWithError(0, "")
		return nil, nil, fmt.Errorf("quic negotiated empty ALPN")
	}
	if conn.ConnectionState().TLS.NegotiatedProtocol != "moq-00" {
		_ = conn.CloseWithError(0, "")
		return nil, nil, fmt.Errorf("unexpected ALPN: %s", conn.ConnectionState().TLS.NegotiatedProtocol)
	}
	return moqConnWithAddr{Connection: quicmoq.NewClient(conn), remoteAddr: conn.RemoteAddr()}, conn.RemoteAddr(), nil
}

