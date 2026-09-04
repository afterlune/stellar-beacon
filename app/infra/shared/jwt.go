package shared

import (
	"benetnasch/app/domain/port"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const jwtIssuer = "红白"

var (
	ed25519PublicKey  ed25519.PublicKey
	ed25519PrivateKey ed25519.PrivateKey
	keyLoadErr        error
)

func init() {
	keyLoadErr = loadOrGenerateKeys()
}

func loadOrGenerateKeys() error {
	privateKeyBase64 := os.Getenv("JWT_PRIVATE_KEY")
	publicKeyBase64 := os.Getenv("JWT_PUBLIC_KEY")
	if privateKeyBase64 == "" || publicKeyBase64 == "" {
		if strings.EqualFold(os.Getenv("JWT_ALLOW_EPHEMERAL"), "true") {
			publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return fmt.Errorf("generate ephemeral JWT key: %w", err)
			}
			ed25519PublicKey = publicKey
			ed25519PrivateKey = privateKey
			return nil
		}
		return errors.New("JWT_PRIVATE_KEY and JWT_PUBLIC_KEY must be configured")
	}

	privateKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return fmt.Errorf("decode JWT private key: %w", err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return fmt.Errorf("decode JWT public key: %w", err)
	}
	if len(privateKey) != ed25519.PrivateKeySize || len(publicKey) != ed25519.PublicKeySize {
		return errors.New("invalid Ed25519 JWT key length")
	}
	ed25519PrivateKey = ed25519.PrivateKey(privateKey)
	ed25519PublicKey = ed25519.PublicKey(publicKey)
	if !ed25519PrivateKey.Public().(ed25519.PublicKey).Equal(ed25519PublicKey) {
		ed25519PrivateKey = nil
		ed25519PublicKey = nil
		return errors.New("JWT public key does not match private key")
	}
	return nil
}

func CreateToken(dto *port.UserDetailsDTO) (string, string, error) {
	return CreateTokenCtx(context.Background(), dto)
}

func CreateTokenCtx(ctx context.Context, dto *port.UserDetailsDTO) (string, string, error) {
	if keyLoadErr != nil {
		return "", "", keyLoadErr
	}
	if err := refreshTokenCtx(ctx, dto); err != nil {
		return "", "", err
	}
	userAuthID := strconv.Itoa(dto.Id)
	now := time.Now()

	accessTokenClaims := jwt2.RegisteredClaims{
		ID:        GetUUID(),
		Subject:   userAuthID,
		Issuer:    jwtIssuer,
		IssuedAt:  jwt2.NewNumericDate(now),
		ExpiresAt: jwt2.NewNumericDate(now.Add(EXPIRE_TIME)),
	}
	accessTokenString, err := signToken(accessTokenClaims)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refreshTokenClaims := jwt2.RegisteredClaims{
		ID:        GetUUID(),
		Subject:   userAuthID,
		Issuer:    jwtIssuer,
		IssuedAt:  jwt2.NewNumericDate(now),
		ExpiresAt: jwt2.NewNumericDate(now.Add(REFRESH_EXPIRE_TIME)),
	}
	refreshTokenString, err := signToken(refreshTokenClaims)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}
	if err := HSetCtx(ctx, REFRESH_TOKEN_PREFIX+userAuthID, refreshTokenString, refreshTokenString, REFRESH_EXPIRE_TIME); err != nil {
		return "", "", fmt.Errorf("store refresh token: %w", err)
	}
	return accessTokenString, refreshTokenString, nil
}

func signToken(claims jwt2.RegisteredClaims) (string, error) {
	token := jwt2.NewWithClaims(jwt2.SigningMethodEdDSA, claims)
	return token.SignedString(ed25519PrivateKey)
}

func refreshTokenCtx(ctx context.Context, dto *port.UserDetailsDTO) error {
	dto.ExpireTime = time.Now().Add(EXPIRE_TIME)
	dto.LastLoginTime = time.Now()
	data, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("marshal user details: %w", err)
	}
	if err := HSetCtx(ctx, LOGIN_USER, strconv.Itoa(dto.Id), data, EXPIRE_TIME); err != nil {
		return fmt.Errorf("store login user: %w", err)
	}
	return nil
}

func GetUUID() string {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(newUUID.String(), "-", "")
}

func TokenParse(tokenStr string) (jwt2.MapClaims, error) {
	return TokenParseCtx(context.Background(), tokenStr)
}

func TokenParseCtx(ctx context.Context, tokenStr string) (jwt2.MapClaims, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}
	if keyLoadErr != nil {
		return nil, keyLoadErr
	}
	if blacklisted, err := IsTokenBlacklistedCtx(ctx, tokenStr); err != nil {
		return nil, err
	} else if blacklisted {
		return nil, errors.New("token is blacklisted")
	}

	token, err := jwt2.Parse(tokenStr, func(token *jwt2.Token) (interface{}, error) {
		if token.Method != jwt2.SigningMethodEdDSA {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ed25519PublicKey, nil
	}, jwt2.WithValidMethods([]string{jwt2.SigningMethodEdDSA.Alg()}), jwt2.WithIssuer(jwtIssuer))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt2.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func InvalidateToken(tokenStr string) error {
	return InvalidateTokenCtx(context.Background(), tokenStr)
}

func InvalidateTokenCtx(ctx context.Context, tokenStr string) error {
	claims, err := parseTokenWithoutValidation(tokenStr)
	if err != nil {
		return err
	}
	ttl := time.Hour
	if exp, ok := claims["exp"].(float64); ok {
		ttl = time.Until(time.Unix(int64(exp), 0))
		if ttl < 0 {
			ttl = 0
		}
	}
	return HSetCtx(ctx, TOKEN_BLACKLIST, tokenStr, "1", ttl)
}

func IsTokenBlacklisted(tokenStr string) bool {
	result, err := IsTokenBlacklistedCtx(context.Background(), tokenStr)
	return err == nil && result
}

func IsTokenBlacklistedCtx(ctx context.Context, tokenStr string) (bool, error) {
	value, err := HGetCtx(ctx, TOKEN_BLACKLIST, tokenStr)
	return value != "", err
}

func RefreshToken(refreshTokenStr string) (string, string, error) {
	return RefreshTokenCtx(context.Background(), refreshTokenStr)
}

func RefreshTokenCtx(ctx context.Context, refreshTokenStr string) (string, string, error) {
	claims, err := TokenParseCtx(ctx, refreshTokenStr)
	if err != nil {
		return "", "", err
	}
	userAuthID, ok := claims["sub"].(string)
	if !ok || userAuthID == "" {
		return "", "", errors.New("invalid refresh token: missing subject")
	}
	storedRefreshToken, err := HGetCtx(ctx, REFRESH_TOKEN_PREFIX+userAuthID, refreshTokenStr)
	if err != nil || storedRefreshToken != refreshTokenStr {
		return "", "", errors.New("invalid refresh token")
	}
	dtoStr, err := HGetCtx(ctx, LOGIN_USER, userAuthID)
	if err != nil || dtoStr == "" {
		return "", "", errors.New("user not found")
	}
	var dto port.UserDetailsDTO
	if err := json.Unmarshal([]byte(dtoStr), &dto); err != nil {
		return "", "", fmt.Errorf("unmarshal user details: %w", err)
	}
	return CreateTokenCtx(ctx, &dto)
}

func parseTokenWithoutValidation(tokenStr string) (jwt2.MapClaims, error) {
	token, _, err := new(jwt2.Parser).ParseUnverified(tokenStr, jwt2.MapClaims{})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt2.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
