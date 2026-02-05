package mcpmoqt

import (
	"crypto/tls"
	"errors"

	"github.com/quic-go/quic-go"
)

type transportRole int

const (
	roleServer transportRole = iota + 1
	roleClient
)

type transportConfig struct {
	role transportRole

	addr string

	tlsServer *tls.Config
	tlsClient *tls.Config

	quicConfig *quic.Config
	alpn       []string
}

type Option func(*transportConfig) error

func defaultALPN() []string { return []string{"moq-00"} }

func WithAddr(addr string) Option {
	return func(c *transportConfig) error {
		c.addr = addr
		return nil
	}
}

func WithQUICConfig(cfg *quic.Config) Option {
	return func(c *transportConfig) error {
		c.quicConfig = cfg
		return nil
	}
}

// WithALPN overrides the default ALPN ("moq-00").
func WithALPN(alpn ...string) Option {
	return func(c *transportConfig) error {
		if len(alpn) == 0 {
			return errors.New("alpn must not be empty")
		}
		c.alpn = append([]string(nil), alpn...)
		return nil
	}
}

// WithTLSServerConfig sets the TLS config used for QUIC listening (server side).
func WithTLSServerConfig(cfg *tls.Config) Option {
	return func(c *transportConfig) error {
		c.tlsServer = cfg
		return nil
	}
}

// WithTLSClientConfig sets the TLS config used for QUIC dialing (client side).
func WithTLSClientConfig(cfg *tls.Config) Option {
	return func(c *transportConfig) error {
		c.tlsClient = cfg
		return nil
	}
}

func applyOptions(role transportRole, opts []Option) (*transportConfig, error) {
	// Defaults: addr localhost:0, ALPN moq-00, QUIC datagrams enabled.
	cfg := &transportConfig{
		role:       role,
		addr:       "localhost:0",
		quicConfig: &quic.Config{EnableDatagrams: true},
		alpn:       defaultALPN(),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	if cfg.addr == "" {
		return nil, errors.New("addr must not be empty")
	}
	return cfg, nil
}
