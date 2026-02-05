package tlsutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

// Local dev helper. Not mandated by draft.
// Generates self-signed certs for local QUIC testing only.
func GenerateSelfSignedServerTLS(alpn []string) (*tls.Config, error) {
	if len(alpn) == 0 {
		alpn = []string{"moq-00"}
	}

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
		NextProtos:   append([]string(nil), alpn...),
	}, nil
}

// EnsureClientTLS returns a usable TLS config for client-side QUIC dialing.
// Default is InsecureSkipVerify with ALPN set for local/dev usage.
func EnsureClientTLS(cfg *tls.Config, alpn []string) *tls.Config {
	if len(alpn) == 0 {
		alpn = []string{"moq-00"}
	}
	if cfg == nil {
		return &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         append([]string(nil), alpn...),
		}
	}
	if len(cfg.NextProtos) == 0 {
		clone := cfg.Clone()
		clone.NextProtos = append([]string(nil), alpn...)
		return clone
	}
	return cfg
}
