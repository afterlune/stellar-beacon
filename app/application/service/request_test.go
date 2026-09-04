package service

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"

	"benetnasch/app/domain/port"
	"github.com/gin-gonic/gin"
)

// serviceTestRequest keeps the service tests framework-backed without making
// production application code depend on Gin. Facade code has the equivalent
// adapter in app/facade/api/request_adapter.go.
type ginContextForServiceTest = gin.Context

type serviceTestRequest struct {
	*ginContextForServiceTest
}

var _ port.Request = serviceTestRequest{}

func (r serviceTestRequest) Context() context.Context {
	if r.ginContextForServiceTest == nil || r.ginContextForServiceTest.Request == nil {
		return context.Background()
	}
	return r.ginContextForServiceTest.Request.Context()
}

func (r serviceTestRequest) Bind(value any) error {
	if r.ginContextForServiceTest == nil {
		return errors.New("request is nil")
	}
	return r.ginContextForServiceTest.ShouldBind(value)
}

func (r serviceTestRequest) BindJSON(value any) error {
	if r.ginContextForServiceTest == nil {
		return errors.New("request is nil")
	}
	return r.ginContextForServiceTest.ShouldBindJSON(value)
}

func (r serviceTestRequest) BindQuery(value any) error {
	if r.ginContextForServiceTest == nil {
		return errors.New("request is nil")
	}
	return r.ginContextForServiceTest.ShouldBindQuery(value)
}

func (r serviceTestRequest) Param(name string) string {
	if r.ginContextForServiceTest == nil {
		return ""
	}
	return r.ginContextForServiceTest.Param(name)
}

func (r serviceTestRequest) Query(name string) string {
	if r.ginContextForServiceTest == nil {
		return ""
	}
	return r.ginContextForServiceTest.Query(name)
}

func (r serviceTestRequest) QueryArray(name string) []string {
	if r.ginContextForServiceTest == nil {
		return nil
	}
	return r.ginContextForServiceTest.QueryArray(name)
}

func (r serviceTestRequest) GetHeader(name string) string {
	if r.ginContextForServiceTest == nil {
		return ""
	}
	return r.ginContextForServiceTest.GetHeader(name)
}

func (r serviceTestRequest) Get(name string) (any, bool) {
	if r.ginContextForServiceTest == nil {
		return nil, false
	}
	return r.ginContextForServiceTest.Get(name)
}

func (r serviceTestRequest) Set(name string, value any) {
	if r.ginContextForServiceTest != nil {
		r.ginContextForServiceTest.Set(name, value)
	}
}

func (r serviceTestRequest) FormFile(name string) (*multipart.FileHeader, error) {
	if r.ginContextForServiceTest == nil {
		return nil, errors.New("request is nil")
	}
	return r.ginContextForServiceTest.FormFile(name)
}

func (r serviceTestRequest) HTTPRequest() *http.Request {
	if r.ginContextForServiceTest == nil {
		return nil
	}
	return r.ginContextForServiceTest.Request
}

func (r serviceTestRequest) ResponseWriter() http.ResponseWriter {
	if r.ginContextForServiceTest == nil {
		return nil
	}
	return r.ginContextForServiceTest.Writer
}
