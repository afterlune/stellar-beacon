package shared

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/zlog"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"github.com/goccy/go-json"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"log"
	"strconv"
	"strings"
	"time"
)

// 全局 Ed25519 密钥对
var (
	ed25519PublicKey  ed25519.PublicKey
	ed25519PrivateKey ed25519.PrivateKey
)

// 初始化 Ed25519 密钥对
func init() {
	var err error
	ed25519PublicKey, ed25519PrivateKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Error generating Ed25519 keys: %v", err)
	}
}

func CreateToken(dto *model.UserDetailsDTO) string {
	refreshToken(dto)
	userAuthId := strconv.Itoa(dto.Id)
	mapCla := jwt2.RegisteredClaims{
		ID:        GetUUID(),
		Subject:   userAuthId,
		Issuer:    "红白",
		IssuedAt:  jwt2.NewNumericDate(time.Now()),
		ExpiresAt: jwt2.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
	}
	token := jwt2.NewWithClaims(jwt2.SigningMethodEdDSA, mapCla)
	tokenString, err := token.SignedString(ed25519PrivateKey)
	if err != nil {
		zlog.Error(err.Error())
	}
	return tokenString
}

func refreshToken(dto *model.UserDetailsDTO) {
	expireTime := time.Now().Add(EXPIRE_TIME)
	dto.ExpireTime = expireTime
	dto.LastLoginTime = time.Now()
	marshal, err := json.Marshal(dto)
	if err != nil {
		zlog.Error(err.Error())
	}
	HSet(LOGIN_USER, strconv.Itoa(dto.Id), marshal, EXPIRE_TIME)
}

func GetUUID() string {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		zlog.Error(err.Error())
	}
	return strings.ReplaceAll(newUUID.String(), "-", "")
}

func TokenParse(tokenStr string) jwt2.MapClaims {
	token, err := jwt2.Parse(tokenStr, func(token *jwt2.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt2.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ed25519PublicKey, nil
	})
	if claims, ok := token.Claims.(jwt2.MapClaims); ok && token.Valid {
		return claims
	} else {
		if err != nil {
			zlog.Error(err.Error())
		}
		return nil
	}
}
