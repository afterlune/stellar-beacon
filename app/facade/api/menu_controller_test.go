package api

import (
	"benetnasch/app/facade/model"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListUserMenusRejectsMissingIdentityWithoutPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/user/menus", ListUserMenus)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/user/menus", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected HTTP status: %d", recorder.Code)
	}
	var result model.ResultVO
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Flag || result.Code != 40001 || result.Message != "用户未登录" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
