package model

import (
	"benetnasch/app/infra/zlog"
	"strconv"
)

type ResultVO struct {
	Flag    bool        `json:"flag"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
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
	QQ_LOGIN_ERROR
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
		9: {"code": "53001", "desc": "qq登录错误"},
	}
	return reInfo[num]
}
func ResultOk() ResultVO {
	info := ResultInfo(SUCCESS)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		zlog.Error(err.Error())
	}
	return ResultVO{Flag: true, Code: code, Message: info["desc"]}
}
func ResultOkWithData(data interface{}) ResultVO {
	info := ResultInfo(SUCCESS)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		zlog.Error(err.Error())
	}
	return ResultVO{Flag: true, Code: code, Message: info["desc"], Data: data}
}
func ResultOkWithDataAndMessage(data interface{}, message string) ResultVO {
	info := ResultInfo(SUCCESS)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		zlog.Error(err.Error())
	}
	return ResultVO{Flag: true, Code: code, Message: message, Data: data}
}
func ResultFail() ResultVO {
	info := ResultInfo(FAIL)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		zlog.Error(err.Error())
	}
	return ResultVO{Flag: false, Code: code, Message: info["desc"]}
}
func ResultFailWithStatus(num int) ResultVO {
	info := ResultInfo(num)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		zlog.Error(err.Error())
	}
	return ResultVO{Flag: false, Code: code, Message: info["desc"]}
}
func ResultFailWithMessage(message string) ResultVO {
	info := ResultInfo(FAIL)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		zlog.Error(err.Error())
	}
	return ResultVO{Code: code, Flag: false, Message: message}
}
func ResultFailWithData(data interface{}) ResultVO {
	info := ResultInfo(FAIL)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		zlog.Error(err.Error())
	}
	return ResultVO{Flag: false, Code: code, Message: info["desc"], Data: data}
}
func ResultFailWithDataAndMessage(data interface{}, message string) ResultVO {
	info := ResultInfo(FAIL)
	code, err := strconv.Atoi(info["code"])
	if err != nil {
		zlog.Error(err.Error())
	}
	return ResultVO{Flag: false, Code: code, Message: message, Data: data}
}
func ResultFailWithCodeAndMessage(code int, message string) ResultVO {
	return ResultVO{Flag: false, Code: code, Message: message}
}

func ResultFailWithCode(code int) ResultVO {
	return ResultVO{Flag: false, Code: code}
}
