package shared

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"benetnasch/app/facade/model"
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
