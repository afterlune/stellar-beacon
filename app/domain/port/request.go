package port

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
)

// Request is the narrow request boundary consumed by application services.
// The application layer must not know which HTTP framework created it.
type Request interface {
	Context() context.Context
	Bind(any) error
	BindJSON(any) error
	BindQuery(any) error
	Param(string) string
	Query(string) string
	QueryArray(string) []string
	GetHeader(string) string
	Get(string) (any, bool)
	Set(string, any)
	FormFile(string) (*multipart.FileHeader, error)
	HTTPRequest() *http.Request
	ResponseWriter() http.ResponseWriter
}

// RequestBody returns the request body when an HTTP body is available. It is
// intentionally kept as a helper so services do not reach into a framework
// context to apply a bounded reader.
func RequestBody(request Request) io.ReadCloser {
	if request == nil || request.HTTPRequest() == nil {
		return nil
	}
	return request.HTTPRequest().Body
}
