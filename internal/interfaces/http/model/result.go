package model

import (
	apperrors "benetnasch/internal/domain/errors"
	"encoding/json"
	"log/slog"
	"strconv"
)

type ResultVO struct {
	// Flag and Code remain internal fields for the application services and
	// tests. MarshalJSON exposes the versioned wire contract below instead.
	Flag    bool        `json:"-"`
	Code    int         `json:"-"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type resultWire struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// MarshalJSON is the single HTTP response envelope. Keeping the legacy
// numeric fields in ResultVO internally lets the existing application
// services migrate incrementally without leaking the old contract.
func (r ResultVO) MarshalJSON() ([]byte, error) {
	return json.Marshal(resultWire{
		Code:    resultCode(r.Code, r.Flag),
		Message: r.Message,
		Data:    r.Data,
	})
}

// UnmarshalJSON accepts both the canonical and pre-migration envelopes. It is
// useful for handler tests and keeps internal middleware diagnostics robust
// while clients move to the canonical string codes.
func (r *ResultVO) UnmarshalJSON(data []byte) error {
	var wire struct {
		Flag    *bool           `json:"flag"`
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	r.Message = wire.Message
	if len(wire.Data) > 0 && string(wire.Data) != "null" {
		if err := json.Unmarshal(wire.Data, &r.Data); err != nil {
			return err
		}
	} else {
		r.Data = nil
	}
	if wire.Flag != nil {
		r.Flag = *wire.Flag
	}
	if len(wire.Code) > 0 {
		var numeric int
		if json.Unmarshal(wire.Code, &numeric) == nil {
			r.Code = numeric
			if wire.Flag == nil {
				r.Flag = numeric == 20000
			}
			return nil
		}
		var code string
		if err := json.Unmarshal(wire.Code, &code); err != nil {
			return err
		}
		r.Code = numericResultCode(code)
		if wire.Flag == nil {
			r.Flag = code == "OK" || code == "SUCCESS"
		}
	}
	return nil
}

func resultCode(code int, flag bool) string {
	if flag || code == 20000 {
		return "OK"
	}
	switch code {
	case 401, 40001, 41000:
		return "UNAUTHENTICATED"
	case 40300:
		return "FORBIDDEN"
	case 40400:
		return "NOT_FOUND"
	case 40900:
		return "CONFLICT"
	case 42900:
		return "RATE_LIMITED"
	case 52000:
		return "INVALID_ARGUMENT"
	case 52001:
		return "USERNAME_EXISTS"
	case 52002:
		return "USERNAME_NOT_FOUND"
	case 52003:
		return "ARTICLE_PASSWORD_INVALID"
	case 51000:
		return "OPERATION_FAILED"
	default:
		return "INTERNAL"
	}
}

func numericResultCode(code string) int {
	switch code {
	case "OK", "SUCCESS":
		return 20000
	case "UNAUTHENTICATED", "AUTH_REQUIRED", "TOKEN_EXPIRED":
		return 40001
	case "FORBIDDEN":
		return 40300
	case "NOT_FOUND":
		return 40400
	case "CONFLICT":
		return 40900
	case "RATE_LIMITED":
		return 42900
	case "INVALID_ARGUMENT":
		return 52000
	case "USERNAME_EXISTS":
		return 52001
	case "USERNAME_NOT_FOUND":
		return 52002
	case "ARTICLE_PASSWORD_INVALID":
		return 52003
	case "OPERATION_FAILED":
		return 51000
	default:
		return 50000
	}
}

const (
	SUCCESS = iota
	NO_LOGIN
	AUTHORIZED
	SYSTEM_ERROR
	FAIL
	VALID_ERROR
	USERNAME_EXIST
	USERNAME_NOT_EXIST
	ARTICLE_ACCESS_FAIL
)

func ResultInfo(num int) map[string]string {
	var reInfo = map[int]map[string]string{
		0: {"code": "20000", "desc": "操作成功"},
		1: {"code": "40001", "desc": "用户未登录"},
		2: {"code": "40300", "desc": "没有操作权限"},
		3: {"code": "50000", "desc": "系统异常"},
		4: {"code": "51000", "desc": "操作失败"},
		5: {"code": "52000", "desc": "参数格式不正确"},
		6: {"code": "52001", "desc": "用户名已存在"},
		7: {"code": "52002", "desc": "用户名不存在"},
		8: {"code": "52003", "desc": "文章密码认证未通过"},
	}
	return reInfo[num]
}
func ResultOk() ResultVO {
	info := ResultInfo(SUCCESS)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		slog.Error("parse result code failed", "error", err)
	}
	return ResultVO{Flag: true, Code: code, Message: info["desc"]}
}
func ResultOkWithData(data interface{}) ResultVO {
	info := ResultInfo(SUCCESS)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		slog.Error("parse result code failed", "error", err)
	}
	return ResultVO{Flag: true, Code: code, Message: info["desc"], Data: data}
}
func ResultOkWithDataAndMessage(data interface{}, message string) ResultVO {
	info := ResultInfo(SUCCESS)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		slog.Error("parse result code failed", "error", err)
	}
	return ResultVO{Flag: true, Code: code, Message: message, Data: data}
}
func ResultFail() ResultVO {
	info := ResultInfo(FAIL)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		slog.Error("parse result code failed", "error", err)
	}
	return ResultVO{Flag: false, Code: code, Message: info["desc"]}
}
func ResultFailWithStatus(num int) ResultVO {
	info := ResultInfo(num)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		slog.Error("parse result code failed", "error", err)
	}
	return ResultVO{Flag: false, Code: code, Message: info["desc"]}
}
func ResultFailWithMessage(message string) ResultVO {
	info := ResultInfo(FAIL)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		slog.Error("parse result code failed", "error", err)
	}
	return ResultVO{Code: code, Flag: false, Message: message}
}
func ResultFailWithData(data interface{}) ResultVO {
	info := ResultInfo(FAIL)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		slog.Error("parse result code failed", "error", err)
	}
	return ResultVO{Flag: false, Code: code, Message: info["desc"], Data: data}
}
func ResultFailWithDataAndMessage(data interface{}, message string) ResultVO {
	info := ResultInfo(FAIL)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		slog.Error("parse result code failed", "error", err)
	}
	return ResultVO{Flag: false, Code: code, Message: message, Data: data}
}
func ResultFailWithCodeAndMessage(code int, message string) ResultVO {
	return ResultVO{Flag: false, Code: code, Message: message}
}

func ResultFailWithCode(code int) ResultVO {
	return ResultVO{Flag: false, Code: code}
}

// ResultFromError keeps the existing response contract while preventing
// infrastructure error details from reaching API clients.
func ResultFromError(err error) ResultVO {
	if err == nil {
		return ResultOk()
	}
	// Keep the public boundary diagnostic free of database, credential, and
	// request details. Lower layers retain the original error for debugging.
	slog.Error("application operation failed",
		"kind", string(apperrors.KindOf(err)),
		"op", apperrors.Op(err),
	)
	switch apperrors.KindOf(err) {
	case apperrors.KindValidation:
		return ResultFailWithMessage("参数格式不正确")
	case apperrors.KindNotFound:
		return ResultFailWithMessage("数据不存在")
	case apperrors.KindUnauthorized:
		return ResultFailWithStatus(NO_LOGIN)
	case apperrors.KindForbidden:
		return ResultFailWithStatus(AUTHORIZED)
	case apperrors.KindConflict:
		return ResultFail()
	default:
		return ResultFailWithMessage("系统繁忙，请稍后再试")
	}
}
