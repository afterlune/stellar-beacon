package api

import (
	"benetnasch/app/application/service"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type apiAgentVitalsProvider struct{}

func (apiAgentVitalsProvider) Snapshot(context.Context) (port.AgentVitals, error) {
	return port.AgentVitals{Phase: port.AgentRhythmAwake, ArticleCount: 3, UniqueVisitorCount: 2}, nil
}

type apiDreamService struct{}

func (apiDreamService) List(context.Context, service.DreamQuery) model.ResultVO {
	return model.ResultOkWithData(model.DreamPageDTO{Records: []model.DreamDTO{{ID: "dream-1"}}, Count: 1})
}

type apiGalaxyService struct {
	called *bool
}

func (s apiGalaxyService) List(context.Context, service.ContentGalaxyQuery) model.ResultVO {
	*s.called = true
	return model.ResultOkWithData(model.ContentGalaxyPageDTO{Count: 1})
}

type apiAgentSafetySwitch struct {
	stopped bool
	err     error
	set     bool
}

func (s *apiAgentSafetySwitch) IsStopped(context.Context) (bool, error) { return s.stopped, s.err }
func (s *apiAgentSafetySwitch) SetStopped(_ context.Context, stopped bool) error {
	s.stopped = stopped
	s.set = true
	return nil
}

func TestGetAgentFeaturesExposesOnlyPublicRolloutFlags(t *testing.T) {
	previous := agentFeatureFlags
	t.Cleanup(func() { agentFeatureFlags = previous })
	agentFeatureFlags = port.AgentFeatureFlags{PublicChat: true, Vitals: false, TTSEnabled: false}
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	GetAgentFeatures(context)
	body := recorder.Body.String()
	if !strings.Contains(body, `"publicChat":true`) || !strings.Contains(body, `"vitals":false`) || !strings.Contains(body, `"ttsEnabled":false`) {
		t.Fatalf("feature response = %q", body)
	}
	if !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("cache policy = %q", recorder.Header().Get("Cache-Control"))
	}
}

func TestGetAgentVitalsIsIndependentlyFeatureGated(t *testing.T) {
	previousFlags := agentFeatureFlags
	previousProvider := agentVitalsProvider
	previousSafety := agentSafetySwitch
	t.Cleanup(func() {
		agentFeatureFlags = previousFlags
		agentVitalsProvider = previousProvider
		agentSafetySwitch = previousSafety
	})
	agentFeatureFlags = port.AgentFeatureFlags{Vitals: true}
	agentVitalsProvider = apiAgentVitalsProvider{}
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/agent/vitals", nil)
	GetAgentVitals(context)
	if !strings.Contains(recorder.Body.String(), `"uniqueVisitorCount":2`) || !strings.Contains(recorder.Body.String(), `"articleCount":3`) {
		t.Fatalf("vitals response = %q", recorder.Body.String())
	}
}

func TestGetDreamsUsesIndependentPublicService(t *testing.T) {
	previousFlags := agentFeatureFlags
	previousService := dreamService
	previousSafety := agentSafetySwitch
	t.Cleanup(func() {
		agentFeatureFlags = previousFlags
		dreamService = previousService
		agentSafetySwitch = previousSafety
	})
	agentFeatureFlags = port.AgentFeatureFlags{Dreams: true}
	dreamService = apiDreamService{}
	agentSafetySwitch = nil
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/dreams", nil)
	GetDreams(context)
	if !strings.Contains(recorder.Body.String(), `"id":"dream-1"`) {
		t.Fatalf("dream response = %q", recorder.Body.String())
	}
}

func TestGetGalaxyDoesNotCallServiceWhenFeatureIsDisabled(t *testing.T) {
	previousFlags := agentFeatureFlags
	previousService := contentGalaxyService
	previousSafety := agentSafetySwitch
	t.Cleanup(func() {
		agentFeatureFlags = previousFlags
		contentGalaxyService = previousService
		agentSafetySwitch = previousSafety
	})
	called := false
	agentFeatureFlags = port.AgentFeatureFlags{}
	contentGalaxyService = apiGalaxyService{called: &called}
	agentSafetySwitch = nil
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/galaxy", nil)
	GetGalaxy(context)
	if called {
		t.Fatal("disabled galaxy feature called the application service")
	}
	if !strings.Contains(recorder.Body.String(), "星河暂未公开") {
		t.Fatalf("disabled galaxy response = %q", recorder.Body.String())
	}
}

func TestAgentEmergencySwitchFailsClosedAtPublicFeatureBoundary(t *testing.T) {
	previousFlags := agentFeatureFlags
	previousSafety := agentSafetySwitch
	t.Cleanup(func() {
		agentFeatureFlags = previousFlags
		agentSafetySwitch = previousSafety
	})
	agentFeatureFlags = port.AgentFeatureFlags{PublicChat: true, Vitals: true, Galaxy: true, Radio: true, Videos: true, TTSEnabled: true}
	agentSafetySwitch = &apiAgentSafetySwitch{stopped: true}
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/agent/features", nil)
	GetAgentFeatures(context)
	if strings.Contains(recorder.Body.String(), `"publicChat":true`) || strings.Contains(recorder.Body.String(), `"vitals":true`) || strings.Contains(recorder.Body.String(), `"galaxy":true`) || strings.Contains(recorder.Body.String(), `"radio":true`) || strings.Contains(recorder.Body.String(), `"videos":true`) || strings.Contains(recorder.Body.String(), `"ttsEnabled":true`) {
		t.Fatalf("stopped feature response exposed enabled flags: %s", recorder.Body.String())
	}
}

func TestSetAgentEmergencyRequiresExplicitBooleanAndPersistsIt(t *testing.T) {
	previousSafety := agentSafetySwitch
	t.Cleanup(func() { agentSafetySwitch = previousSafety })
	switcher := &apiAgentSafetySwitch{}
	agentSafetySwitch = switcher
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/admin/agent/emergency", strings.NewReader(`{"stopped":true}`))
	context.Request.Header.Set("Content-Type", "application/json")
	SetAgentEmergency(context)
	if !switcher.set || !switcher.stopped || !strings.Contains(recorder.Body.String(), `"stopped":true`) {
		t.Fatalf("emergency response=%s switch=%+v", recorder.Body.String(), switcher)
	}

	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/admin/agent/emergency", strings.NewReader(`{}`))
	context.Request.Header.Set("Content-Type", "application/json")
	SetAgentEmergency(context)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing stopped status = %d, want 400", recorder.Code)
	}
}
