package shared

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/zlog"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/goccy/go-json"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"strconv"
	"strings"
	"time"
)

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
	tokenString, err := token.SignedString([]byte(generalKey()))
	if err != nil {
		zlog.Error(err.Error())
	}
	return tokenString
}

func generalKey() string {
	encodedKey := base64.StdEncoding.EncodeToString([]byte(SECRET))
	h := hmac.New(sha256.New, []byte(SECRET))
	h.Write([]byte(encodedKey))
	return hex.EncodeToString(h.Sum(nil))
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
		return []byte(generalKey()), nil
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
