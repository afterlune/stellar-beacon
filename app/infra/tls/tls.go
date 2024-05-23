package tls

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/bwmarrin/snowflake"
	rand2 "golang.org/x/exp/rand"
	"math/big"
	"os"
	"sync"
)

func GenerateTLSConfig(prefix int, host string) *tls.Config {
	switch prefix {
	case 0:
		return newServerTLSConfig("", "", "")
	case 1:
		return newClientTLSConfig("", "", "", host)
	}
	return nil
}

var (
	once sync.Once
	node *snowflake.Node
)

func GenerateID() snowflake.ID {
	once.Do(func() {
		nod, err := snowflake.NewNode(rand2.Int63n(1023)) // 传入节点ID
		if err != nil {
			return
		}
		node = nod
	})
	return node.Generate()
}

func newCertificate() tls.Certificate {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{SerialNumber: big.NewInt(GenerateID().Int64())}
	crtDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		publicKey,
		privateKey)
	if err != nil {
		panic(err)
	}
	key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		panic(err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "ED25519 PRIVATE KEY", Bytes: key})
	crtPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: crtDER})

	tlsCert, err := tls.X509KeyPair(crtPEM, keyPEM)
	if err != nil {
		panic(err)
	}

	tlsCert.SupportedSignatureAlgorithms = []tls.SignatureScheme{tls.Ed25519}

	return tlsCert
}

func newServerTLSConfig(certPath, keyPath, caPath string) *tls.Config {
	base := &tls.Config{MinVersion: tls.VersionTLS13, CipherSuites: []uint16{tls.TLS_CHACHA20_POLY1305_SHA256}}

	if certPath == "" || keyPath == "" {
		cert := newCertificate()
		base.Certificates = []tls.Certificate{cert}
	} else {
		cert, err := newTLSKey(certPath, keyPath)
		if err != nil {
			return nil
		}

		base.Certificates = []tls.Certificate{*cert}
	}

	if caPath != "" {
		pool, err := newCertPool(caPath)
		if err != nil {
			return nil
		}

		base.ClientAuth = tls.RequireAndVerifyClientCert
		base.ClientCAs = pool
	}
	return base
}

func newClientTLSConfig(certPath, keyPath, caPath, serverName string) *tls.Config {
	base := &tls.Config{MinVersion: tls.VersionTLS13, CipherSuites: []uint16{tls.TLS_CHACHA20_POLY1305_SHA256}}

	if certPath != "" && keyPath != "" {
		cert, err := newTLSKey(certPath, keyPath)
		if err != nil {
			return nil
		}

		base.Certificates = []tls.Certificate{*cert}
	}

	base.ServerName = serverName

	if caPath != "" {
		pool, err := newCertPool(caPath)
		if err != nil {
			return nil
		}

		base.RootCAs = pool
		base.InsecureSkipVerify = false
	} else {
		base.InsecureSkipVerify = true
	}
	return base
}

func newTLSKey(certfile, keyfile string) (*tls.Certificate, error) {
	tlsCert, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

func newCertPool(caPath string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()

	caCrt, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	pool.AppendCertsFromPEM(caCrt)

	return pool, nil
}
