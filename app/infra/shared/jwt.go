package shared

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/zlog"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// 全局 Ed25519 密钥对
var (
	ed25519PublicKey  ed25519.PublicKey
	ed25519PrivateKey ed25519.PrivateKey
)

// 初始化 Ed25519 密钥对
func init() {
	loadOrGenerateKeys()
}

// 加载或生成密钥
func loadOrGenerateKeys() {
	// 尝试从环境变量加载密钥
	privateKeyBase64 := os.Getenv("JWT_PRIVATE_KEY")
	publicKeyBase64 := os.Getenv("JWT_PUBLIC_KEY")

	if privateKeyBase64 != "" && publicKeyBase64 != "" {
		// 解析已有密钥
		privateKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
		if err != nil {
			zlog.Error("Failed to decode private key: " + err.Error())
			generateAndSaveKeys()
			return
		}

		publicKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
		if err != nil {
			zlog.Error("Failed to decode public key: " + err.Error())
			generateAndSaveKeys()
			return
		}

		ed25519PrivateKey = ed25519.PrivateKey(privateKey)
		ed25519PublicKey = ed25519.PublicKey(publicKey)
		zlog.Info("Loaded JWT keys from environment variables")
	} else {
		// 生成新密钥
		generateAndSaveKeys()
	}
}

// 生成并保存密钥
func generateAndSaveKeys() {
	var err error
	ed25519PublicKey, ed25519PrivateKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Error generating Ed25519 keys: %v", err)
	}

	// 编码为base64并输出（实际生产环境中应该存储到安全的地方）
	privateKeyBase64 := base64.StdEncoding.EncodeToString(ed25519PrivateKey)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(ed25519PublicKey)

	zlog.Info("Generated new JWT keys")
	zlog.Debug("Private key: " + privateKeyBase64)
	zlog.Debug("Public key: " + publicKeyBase64)
}

// CreateToken 创建访问令牌
func CreateToken(dto *model.UserDetailsDTO) (string, string, error) {
	refreshToken(dto)
	userAuthId := strconv.Itoa(dto.Id)

	// 创建访问令牌
	accessTokenClaims := jwt2.RegisteredClaims{
		ID:        GetUUID(),
		Subject:   userAuthId,
		Issuer:    "红白",
		IssuedAt:  jwt2.NewNumericDate(time.Now()),
		ExpiresAt: jwt2.NewNumericDate(time.Now().Add(EXPIRE_TIME)),
	}
	accessToken := jwt2.NewWithClaims(jwt2.SigningMethodEdDSA, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString(ed25519PrivateKey)
	if err != nil {
		zlog.Error("Failed to sign access token: " + err.Error())
		return "", "", err
	}

	// 创建刷新令牌
	refreshTokenClaims := jwt2.RegisteredClaims{
		ID:        GetUUID(),
		Subject:   userAuthId,
		Issuer:    "红白",
		IssuedAt:  jwt2.NewNumericDate(time.Now()),
		ExpiresAt: jwt2.NewNumericDate(time.Now().Add(REFRESH_EXPIRE_TIME)),
	}
	refreshToken := jwt2.NewWithClaims(jwt2.SigningMethodEdDSA, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString(ed25519PrivateKey)
	if err != nil {
		zlog.Error("Failed to sign refresh token: " + err.Error())
		return "", "", err
	}

	// 存储刷新令牌
	success := HSet(REFRESH_TOKEN_PREFIX+userAuthId, refreshTokenString, refreshTokenString, REFRESH_EXPIRE_TIME)
	if !success {
		zlog.Error("Failed to store refresh token")
		return "", "", fmt.Errorf("failed to store refresh token")
	}

	return accessTokenString, refreshTokenString, nil
}

func refreshToken(dto *model.UserDetailsDTO) {
	expireTime := time.Now().Add(EXPIRE_TIME)
	dto.ExpireTime = expireTime
	dto.LastLoginTime = time.Now()
	marshal, err := json.Marshal(dto)
	if err != nil {
		zlog.Error("Failed to marshal user details: " + err.Error())
	}
	HSet(LOGIN_USER, strconv.Itoa(dto.Id), marshal, EXPIRE_TIME)
}

func GetUUID() string {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		zlog.Error("Failed to generate UUID: " + err.Error())
	}
	return strings.ReplaceAll(newUUID.String(), "-", "")
}

// TokenParse 解析并验证令牌
func TokenParse(tokenStr string) (jwt2.MapClaims, error) {
	// 检查令牌是否在黑名单中
	if IsTokenBlacklisted(tokenStr) {
		return nil, fmt.Errorf("token is blacklisted")
	}

	token, err := jwt2.Parse(tokenStr, func(token *jwt2.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt2.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ed25519PublicKey, nil
	})

	if err != nil {
		zlog.Error("Failed to parse token: " + err.Error())
		return nil, err
	}

	if claims, ok := token.Claims.(jwt2.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("invalid token")
	}
}

// InvalidateToken 将令牌加入黑名单
func InvalidateToken(tokenStr string) error {
	// 解析令牌获取过期时间
	claims, err := parseTokenWithoutValidation(tokenStr)
	if err != nil {
		return err
	}

	// 计算过期时间
	var ttl time.Duration
	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		ttl = time.Until(expTime)
		if ttl < 0 {
			ttl = 0
		}
	} else {
		// 如果没有过期时间，设置默认TTL为1小时
		ttl = time.Hour
	}

	// 添加到黑名单
	success := HSet(TOKEN_BLACKLIST, tokenStr, "1", ttl)
	if !success {
		zlog.Error("Failed to add token to blacklist")
		return fmt.Errorf("failed to add token to blacklist")
	}

	zlog.Info("Token added to blacklist")
	return nil
}

// IsTokenBlacklisted 检查令牌是否在黑名单中
func IsTokenBlacklisted(tokenStr string) bool {
	result := HGet(TOKEN_BLACKLIST, tokenStr)
	return result != ""
}

// RefreshToken 使用刷新令牌获取新的访问令牌
func RefreshToken(refreshTokenStr string) (string, string, error) {
	// 解析刷新令牌
	claims, err := TokenParse(refreshTokenStr)
	if err != nil {
		return "", "", err
	}

	// 获取用户ID
	userAuthId, ok := claims["sub"].(string)
	if !ok || userAuthId == "" {
		return "", "", fmt.Errorf("invalid refresh token: missing subject")
	}

	// 验证刷新令牌是否与存储的一致
	storedRefreshToken := HGet(REFRESH_TOKEN_PREFIX+userAuthId, refreshTokenStr)
	if storedRefreshToken != refreshTokenStr {
		return "", "", fmt.Errorf("invalid refresh token")
	}

	// 从Redis获取用户信息
	dtoStr := HGet(LOGIN_USER, userAuthId)
	if dtoStr == "" {
		return "", "", fmt.Errorf("user not found")
	}

	var userDetailsDTO model.UserDetailsDTO
	if err := json.Unmarshal([]byte(dtoStr), &userDetailsDTO); err != nil {
		zlog.Error("Failed to unmarshal user details: " + err.Error())
		return "", "", err
	}

	// 生成新的访问令牌和刷新令牌
	return CreateToken(&userDetailsDTO)
}

// parseTokenWithoutValidation 解析令牌但不验证签名
func parseTokenWithoutValidation(tokenStr string) (jwt2.MapClaims, error) {
	token, _, err := new(jwt2.Parser).ParseUnverified(tokenStr, jwt2.MapClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt2.MapClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}
