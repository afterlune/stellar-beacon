package api

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"

	"benetnasch/app/domain/port"
	"github.com/gin-gonic/gin"
)

// ginRequest is the only Gin-to-application adapter. Application services
// consume port.Request and therefore cannot depend on Gin or another HTTP
// framework.
type ginRequest struct {
	context *gin.Context
}

func applicationRequest(context *gin.Context) port.Request {
	if context == nil {
		return nil
	}
	return ginRequest{context: context}
}

func (r ginRequest) Context() context.Context {
	if r.context == nil || r.context.Request == nil {
		return context.Background()
	}
	return r.context.Request.Context()
}

func (r ginRequest) Bind(value any) error {
	if r.context == nil {
		return errors.New("request is nil")
	}
	return r.context.ShouldBind(value)
}

func (r ginRequest) BindJSON(value any) error {
	if r.context == nil {
		return errors.New("request is nil")
	}
	return r.context.ShouldBindJSON(value)
}

func (r ginRequest) BindQuery(value any) error {
	if r.context == nil {
		return errors.New("request is nil")
	}
	return r.context.ShouldBindQuery(value)
}

func (r ginRequest) Param(name string) string {
	if r.context == nil {
		return ""
	}
	return r.context.Param(name)
}

func (r ginRequest) Query(name string) string {
	if r.context == nil {
		return ""
	}
	return r.context.Query(name)
}

func (r ginRequest) QueryArray(name string) []string {
	if r.context == nil {
		return nil
	}
	return r.context.QueryArray(name)
}

func (r ginRequest) GetHeader(name string) string {
	if r.context == nil {
		return ""
	}
	return r.context.GetHeader(name)
}

func (r ginRequest) Get(name string) (any, bool) {
	if r.context == nil {
		return nil, false
	}
	return r.context.Get(name)
}

func (r ginRequest) Set(name string, value any) {
	if r.context != nil {
		r.context.Set(name, value)
	}
}

func (r ginRequest) FormFile(name string) (*multipart.FileHeader, error) {
	if r.context == nil {
		return nil, errors.New("request is nil")
	}
	return r.context.FormFile(name)
}

func (r ginRequest) HTTPRequest() *http.Request {
	if r.context == nil {
		return nil
	}
	return r.context.Request
}

func (r ginRequest) ResponseWriter() http.ResponseWriter {
	if r.context == nil {
		return nil
	}
	return r.context.Writer
}
