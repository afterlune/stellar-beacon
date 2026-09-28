package shared

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
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
	if privateKeyBase64 != "" || publicKeyBase64 != "" {
		if privateKeyBase64 == "" || publicKeyBase64 == "" {
			return errors.New("JWT_PRIVATE_KEY and JWT_PUBLIC_KEY must be configured together")
		}
		return loadEncodedKeys(privateKeyBase64, publicKeyBase64)
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("JWT_ALLOW_EPHEMERAL")), "true") {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return fmt.Errorf("generate ephemeral JWT key: %w", err)
		}
		ed25519PublicKey = publicKey
		ed25519PrivateKey = privateKey
		return nil
	}

	keyDirectory := strings.TrimSpace(os.Getenv("JWT_KEY_DIR"))
	if keyDirectory == "" {
		return errors.New("JWT key pair or JWT_KEY_DIR must be configured")
	}
	return loadPersistentKeys(keyDirectory)
}

func loadEncodedKeys(privateKeyBase64, publicKeyBase64 string) error {
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return fmt.Errorf("decode JWT private key: %w", err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return fmt.Errorf("decode JWT public key: %w", err)
	}
	return setKeys(privateKey, publicKey)
}

func setKeys(privateKeyBytes, publicKeyBytes []byte) error {
	if len(privateKeyBytes) != ed25519.PrivateKeySize || len(publicKeyBytes) != ed25519.PublicKeySize {
		return errors.New("invalid Ed25519 JWT key length")
	}
	ed25519PrivateKey = ed25519.PrivateKey(privateKeyBytes)
	ed25519PublicKey = ed25519.PublicKey(publicKeyBytes)
	if !ed25519PrivateKey.Public().(ed25519.PublicKey).Equal(ed25519PublicKey) {
		ed25519PrivateKey = nil
		ed25519PublicKey = nil
		return errors.New("JWT public key does not match private key")
	}
	return nil
}

func loadPersistentKeys(directory string) error {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create JWT key directory: %w", err)
	}
	privatePath := filepath.Join(directory, "ed25519-private.key")
	publicPath := filepath.Join(directory, "ed25519-public.key")
	privateKey, privateErr := os.ReadFile(privatePath)
	publicKey, publicErr := os.ReadFile(publicPath)
	if privateErr == nil || publicErr == nil {
		if privateErr != nil || publicErr != nil {
			return errors.New("JWT key directory contains an incomplete key pair")
		}
		if runtime.GOOS != "windows" {
			privateInfo, err := os.Stat(privatePath)
			if err != nil {
				return fmt.Errorf("stat JWT private key: %w", err)
			}
			if privateInfo.Mode().Perm()&0o077 != 0 {
				return errors.New("JWT private key permissions must not allow group or other access")
			}
		}
		return setKeys(privateKey, publicKey)
	}
	if !errors.Is(privateErr, os.ErrNotExist) || !errors.Is(publicErr, os.ErrNotExist) {
		return fmt.Errorf("read JWT key pair: %w", errors.Join(privateErr, publicErr))
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate persistent JWT key: %w", err)
	}
	if err := writePrivateKey(privatePath, privateKey); err != nil {
		return err
	}
	if err := os.WriteFile(publicPath, publicKey, 0o644); err != nil {
		_ = os.Remove(privatePath)
		return fmt.Errorf("write JWT public key: %w", err)
	}
	return setKeys(privateKey, publicKey)
}

func writePrivateKey(path string, privateKey ed25519.PrivateKey) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("JWT key pair was created concurrently; restart the process: %w", err)
		}
		return fmt.Errorf("create JWT private key: %w", err)
	}
	if _, err := file.Write(privateKey); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write JWT private key: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("sync JWT private key: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close JWT private key: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("set JWT private key permissions: %w", err)
	}
	return nil
}

// ValidateJWTKeys makes an invalid or non-persistent key setup fail before the
// server begins accepting requests.
func ValidateJWTKeys() error {
	if keyLoadErr != nil {
		return keyLoadErr
	}
	if len(ed25519PrivateKey) != ed25519.PrivateKeySize || len(ed25519PublicKey) != ed25519.PublicKeySize {
		return errors.New("JWT key pair is not initialized")
	}
	return nil
}

func CreateTokenCtx(ctx context.Context, dto *entity.AuthSession) (string, string, error) {
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

func refreshTokenCtx(ctx context.Context, dto *entity.AuthSession) error {
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

func IsTokenBlacklistedCtx(ctx context.Context, tokenStr string) (bool, error) {
	value, err := HGetCtx(ctx, TOKEN_BLACKLIST, tokenStr)
	return value != "", err
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
	var dto entity.AuthSession
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
