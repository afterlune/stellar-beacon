package shared

import (
	md52 "crypto/md5"
	"encoding/hex"
)

func GetMD5(s string) string {
	data := []byte(s)
	md5 := md52.New()
	md5.Write(data)
	return hex.EncodeToString(md5.Sum(nil))
}
