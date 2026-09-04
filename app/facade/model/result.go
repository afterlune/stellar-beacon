package model

import "benetnasch/app/domain/port"

type ResultVO = port.ResultVO

const (
	SUCCESS             = port.SUCCESS
	NO_LOGIN            = port.NO_LOGIN
	AUTHORIZED          = port.AUTHORIZED
	SYSTEM_ERROR        = port.SYSTEM_ERROR
	FAIL                = port.FAIL
	VALID_ERROR         = port.VALID_ERROR
	USERNAME_EXIST      = port.USERNAME_EXIST
	USERNAME_NOT_EXIST  = port.USERNAME_NOT_EXIST
	ARTICLE_ACCESS_FAIL = port.ARTICLE_ACCESS_FAIL
)

func ResultInfo(num int) map[string]string       { return port.ResultInfo(num) }
func ResultOk() ResultVO                         { return port.ResultOk() }
func ResultOkWithData(data interface{}) ResultVO { return port.ResultOkWithData(data) }
func ResultOkWithDataAndMessage(data interface{}, message string) ResultVO {
	return port.ResultOkWithDataAndMessage(data, message)
}
func ResultFail() ResultVO                          { return port.ResultFail() }
func ResultFailWithStatus(num int) ResultVO         { return port.ResultFailWithStatus(num) }
func ResultFailWithMessage(message string) ResultVO { return port.ResultFailWithMessage(message) }
func ResultFailWithData(data interface{}) ResultVO  { return port.ResultFailWithData(data) }
func ResultFailWithDataAndMessage(data interface{}, message string) ResultVO {
	return port.ResultFailWithDataAndMessage(data, message)
}
func ResultFailWithCodeAndMessage(code int, message string) ResultVO {
	return port.ResultFailWithCodeAndMessage(code, message)
}
func ResultFailWithCode(code int) ResultVO { return port.ResultFailWithCode(code) }
func ResultFromError(err error) ResultVO   { return port.ResultFromError(err) }
