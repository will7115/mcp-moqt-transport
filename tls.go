package mcpmoqt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"time"
)

// SelfSignedTLSServerConfig returns a minimal TLS config suitable for local QUIC
// tests. It includes "moq-00" in NextProtos.
func SelfSignedTLSServerConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:              []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"moq-00", "h3"},
	}, nil
}

func ensureServerTLS(cfg *transportConfig) error {
	if cfg.tlsServer == nil {
		return errors.New("server tls config is required (use WithTLSServerConfig)")
	}
	if len(cfg.tlsServer.NextProtos) == 0 {
		cfg.tlsServer = cfg.tlsServer.Clone()
		cfg.tlsServer.NextProtos = append([]string(nil), cfg.alpn...)
	}
	return nil
}

func ensureClientTLS(cfg *transportConfig) error {
	if cfg.tlsClient == nil {
		// Default to insecure for local/dev usage; users can override via WithTLSClientConfig.
		cfg.tlsClient = &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         append([]string(nil), cfg.alpn...),
		}
		return nil
	}
	if len(cfg.tlsClient.NextProtos) == 0 {
		cfg.tlsClient = cfg.tlsClient.Clone()
		cfg.tlsClient.NextProtos = append([]string(nil), cfg.alpn...)
	}
	return nil
}

