package mcpmoqt

import (
	"crypto/tls"

	"github.com/mcp-moqt/mcp-moqt-transport/internal/tlsutil"
)

// SelfSignedTLSServerConfig returns a minimal TLS config suitable for local QUIC
// tests. It includes "moq-00" in NextProtos.
func SelfSignedTLSServerConfig() (*tls.Config, error) {
	return tlsutil.GenerateSelfSignedServerTLS(defaultALPN())
}

func ensureServerTLS(cfg *transportConfig) error {
	if cfg.tlsServer == nil {
		tlsServer, err := tlsutil.GenerateSelfSignedServerTLS(cfg.alpn)
		if err != nil {
			return err
		}
		cfg.tlsServer = tlsServer
		return nil
	}
	if len(cfg.tlsServer.NextProtos) == 0 {
		cfg.tlsServer = cfg.tlsServer.Clone()
		cfg.tlsServer.NextProtos = append([]string(nil), cfg.alpn...)
	}
	return nil
}

func ensureClientTLS(cfg *transportConfig) error {
	cfg.tlsClient = tlsutil.EnsureClientTLS(cfg.tlsClient, cfg.alpn)
	return nil
}
