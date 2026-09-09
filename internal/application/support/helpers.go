package support

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

func CheckEmail(value string) bool {
	match, err := regexp.MatchString(`\w[-\w.+]*@([A-Za-z0-9][-A-Za-z0-9]+\.)+[A-Za-z]{2,14}`, value)
	return err == nil && match
}

func RandomCode() string {
	return fmt.Sprintf("%06d", rand.Intn(900000)+100000)
}

func StructCopy(old, new any) error {
	data, err := json.Marshal(old)
	if err != nil {
		return fmt.Errorf("marshal value for struct copy: %w", err)
	}
	if err := json.Unmarshal(data, new); err != nil {
		return fmt.Errorf("unmarshal value for struct copy: %w", err)
	}
	return nil
}

func Unmarsh(old any, new any) error {
	value, ok := old.(string)
	if !ok {
		return fmt.Errorf("expected cached JSON string, got %T", old)
	}
	return json.Unmarshal([]byte(value), new)
}

func GetMD5(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func GetUUID() string {
	id, err := uuid.NewUUID()
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(id.String(), "-", "")
}

func ObjectKey(filename, prefix string) (string, error) {
	index := strings.LastIndex(filename, ".")
	if index <= 0 || index == len(filename)-1 {
		return "", errors.New("filename must contain a non-empty extension")
	}
	name := GetMD5(filename[:index]) + filename[index:]
	return strings.TrimRight(prefix, "/") + "/" + name, nil
}
