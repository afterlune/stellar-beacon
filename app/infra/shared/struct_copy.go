package shared

import (
	"benetnasch/app/infra/zlog"
	"github.com/goccy/go-json"
)

func StructCopy(old, new interface{}) {
	marshal, err := json.Marshal(old)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	err = json.Unmarshal(marshal, new)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
}
func Unmarsh(old, new interface{}) {
	err := json.Unmarshal([]byte(old.(string)), new)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
}
