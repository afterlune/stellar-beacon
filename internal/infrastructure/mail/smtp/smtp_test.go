package mailer

import (
	"context"
	"net"
	"testing"

	"github.com/afterlune/stellar-beacon/internal/infrastructure/config"
)

func TestSMTPMailerCheckPlaintextEndpoint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	conf := &config.Email{
		EmailAccount: "integration@example.test",
		SmtpName:     "127.0.0.1",
		SmtpPort:     listener.Addr().(*net.TCPAddr).Port,
		Auth:         false,
		TLS:          false,
	}
	status := NewSMTPMailer(conf).Check(context.Background())
	if !status.Configured || !status.Reachable {
		t.Fatalf("SMTP health = %+v", status)
	}
	if status.TLS || status.Auth {
		t.Fatalf("plaintext health reported secure transport: %+v", status)
	}
}
