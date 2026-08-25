package api

import (
	"benetnasch/app/facade/model"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListArticlesKeepsResultEnvelopeOnInvalidPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/articles/all", ListArticles)
	req := httptest.NewRequest(http.MethodGet, "/articles/all?current=invalid&size=10", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected HTTP status: %d", recorder.Code)
	}
	var result model.ResultVO
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
