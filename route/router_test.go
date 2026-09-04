package route

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouterSetupRegistersAgentMemoryAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RouterSetup(router)

	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{
		"GET /internal/space/v1/capabilities",
		"POST /internal/space/v1/search",
		"GET /internal/space/v1/content/:type/:id",
		"POST /internal/space/v1/publications",
		"POST /admin/ai/vision/preview",
		"GET /admin/ai/observability",
		"GET /admin/ai/memory/assertions",
		"GET /admin/ai/memory/assertions/:id/history",
		"DELETE /admin/ai/memory/assertions/:id",
		"GET /admin/ai/memory/conflicts",
		"POST /admin/ai/memory/conflicts/:id/resolve",
		"POST /admin/ai/memory/conflicts/:id/reject",
	} {
		if _, ok := registered[expected]; !ok {
			t.Errorf("route %s is not registered", expected)
		}
	}
}
