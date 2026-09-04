package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"benetnasch/app/application/service"
	"benetnasch/app/domain/port"

	"github.com/gin-gonic/gin"
)

type apiOperationalObservabilityService struct {
	result port.ResultVO
}

func (s apiOperationalObservabilityService) Get(context.Context) port.ResultVO {
	return s.result
}

func TestGetAIOperationalMetricsReturnsServiceSnapshot(t *testing.T) {
	previous := operationalObservability
	t.Cleanup(func() { operationalObservability = previous })
	operationalObservability = apiOperationalObservabilityService{
		result: port.ResultOkWithData(port.OperationalMetricsSnapshot{
			AI:     port.AIOperationalSnapshot{Enabled: true},
			Search: port.SearchMetricsSnapshot{DurationP95MS: 42},
		}),
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/ai/observability", nil)

	GetAIOperationalMetrics(c)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"durationP95Ms":42`) || !strings.Contains(recorder.Body.String(), `"enabled":true`) {
		t.Fatalf("observability response status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

var _ service.OperationalObservabilityService = apiOperationalObservabilityService{}
