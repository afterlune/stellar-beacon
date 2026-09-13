package shared

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
)

func TestTokenRoundTripAndRefreshTokenStorage(t *testing.T) {
	_, cleanup := newTestRedis(t)
	defer cleanup()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	oldPublicKey, oldPrivateKey, oldKeyErr := ed25519PublicKey, ed25519PrivateKey, keyLoadErr
	ed25519PublicKey, ed25519PrivateKey, keyLoadErr = publicKey, privateKey, nil
	defer func() {
		ed25519PublicKey, ed25519PrivateKey, keyLoadErr = oldPublicKey, oldPrivateKey, oldKeyErr
	}()

	dto := &model.UserDetailsDTO{Id: 42, Username: "test@example.com"}
	accessToken, refreshToken, err := CreateTokenCtx(context.Background(), dto)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := TokenParseCtx(context.Background(), accessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != "42" || claims["iss"] != jwtIssuer {
		t.Fatalf("claims = %#v, want subject 42 and issuer %q", claims, jwtIssuer)
	}
	stored, err := HGetCtx(context.Background(), REFRESH_TOKEN_PREFIX+"42", refreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if stored != refreshToken {
		t.Fatalf("stored refresh token does not match generated token")
	}
}

func TestPersistentJWTKeyPairSurvivesReload(t *testing.T) {
	oldPublicKey, oldPrivateKey, oldKeyErr := ed25519PublicKey, ed25519PrivateKey, keyLoadErr
	defer func() {
		ed25519PublicKey, ed25519PrivateKey, keyLoadErr = oldPublicKey, oldPrivateKey, oldKeyErr
	}()

	keyDirectory := t.TempDir()
	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PUBLIC_KEY", "")
	t.Setenv("JWT_ALLOW_EPHEMERAL", "false")
	t.Setenv("JWT_KEY_DIR", keyDirectory)
	if err := loadOrGenerateKeys(); err != nil {
		t.Fatalf("initialize persistent key pair: %v", err)
	}
	firstPrivate := append(ed25519.PrivateKey(nil), ed25519PrivateKey...)
	firstPublic := append(ed25519.PublicKey(nil), ed25519PublicKey...)
	if err := loadOrGenerateKeys(); err != nil {
		t.Fatalf("reload persistent key pair: %v", err)
	}
	if !bytes.Equal(ed25519PrivateKey, firstPrivate) || !bytes.Equal(ed25519PublicKey, firstPublic) {
		t.Fatal("JWT key pair changed after reloading the persistent directory")
	}

	privateInfo, err := os.Stat(filepath.Join(keyDirectory, "ed25519-private.key"))
	if err != nil {
		t.Fatalf("stat persisted private key: %v", err)
	}
	if runtime.GOOS != "windows" && privateInfo.Mode().Perm() != 0o600 {
		t.Fatalf("private key permissions = %o, want 600", privateInfo.Mode().Perm())
	}
}

func TestPersistentJWTKeysRejectAnIncompletePair(t *testing.T) {
	keyDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(keyDirectory, "ed25519-private.key"), make([]byte, ed25519.PrivateKeySize), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := loadPersistentKeys(keyDirectory); err == nil {
		t.Fatal("loadPersistentKeys() accepted an incomplete pair")
	}
}
