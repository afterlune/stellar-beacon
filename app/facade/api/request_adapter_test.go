package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestApplicationRequestAdaptsGinRequestAndContextState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/test/42?tag=a&tag=b", strings.NewReader(`{"name":"test"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "id", Value: "42"}}
	context.Set("actor", "admin")

	request := applicationRequest(context)
	var body struct {
		Name string `json:"name"`
	}
	if err := request.BindJSON(&body); err != nil {
		t.Fatal(err)
	}
	if body.Name != "test" || request.Param("id") != "42" || strings.Join(request.QueryArray("tag"), ",") != "a,b" {
		t.Fatalf("adapted request lost input: body=%+v id=%q tags=%v", body, request.Param("id"), request.QueryArray("tag"))
	}
	if actor, ok := request.Get("actor"); !ok || actor != "admin" {
		t.Fatalf("adapted request lost context value: %v, %v", actor, ok)
	}
	request.Set("trace", "present")
	if value, ok := context.Get("trace"); !ok || value != "present" {
		t.Fatalf("adapted request did not write context value: %v, %v", value, ok)
	}
	if request.HTTPRequest() != context.Request || request.ResponseWriter() != context.Writer {
		t.Fatal("adapted request did not expose the original transport objects")
	}
}

func TestApplicationRequestNilIsSafe(t *testing.T) {
	if request := applicationRequest(nil); request != nil {
		t.Fatal("nil Gin context must not produce a non-nil application request")
	}
}
