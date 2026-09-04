package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"benetnasch/app/application/service"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"

	"github.com/gin-gonic/gin"
)

type apiRadioServiceFake struct{}

func (apiRadioServiceFake) Current(context.Context) model.ResultVO {
	return model.ResultOkWithData(model.RadioPageDTO{Count: 1})
}

type apiVideoServiceFake struct {
	called *bool
}

func (s apiVideoServiceFake) markCalled() {
	if s.called != nil {
		*s.called = true
	}
}

func (s apiVideoServiceFake) List(context.Context, service.VideoQuery) model.ResultVO {
	s.markCalled()
	return model.ResultOkWithData(model.VideoPageDTO{})
}

func (s apiVideoServiceFake) Upload(context.Context, service.VideoUploadInput) model.ResultVO {
	s.markCalled()
	return model.ResultOk()
}

func (s apiVideoServiceFake) CreateExternal(context.Context, service.ExternalVideoInput) model.ResultVO {
	s.markCalled()
	return model.ResultOk()
}

func (s apiVideoServiceFake) Delete(context.Context, string) model.ResultVO {
	s.markCalled()
	return model.ResultOk()
}

func (apiVideoServiceFake) AllowedFrameOrigins() []string {
	return []string{"https://video.example"}
}

func TestMediaControllersApplyIndependentFeatureFlagsAndVideoCSP(t *testing.T) {
	previousFlags := agentFeatureFlags
	previousSafety := agentSafetySwitch
	previousRadio := radioService
	previousVideos := videoService
	t.Cleanup(func() {
		agentFeatureFlags = previousFlags
		agentSafetySwitch = previousSafety
		radioService = previousRadio
		videoService = previousVideos
	})
	agentFeatureFlags = port.AgentFeatureFlags{Radio: true, Videos: true}
	agentSafetySwitch = nil
	radioService = apiRadioServiceFake{}
	videoService = apiVideoServiceFake{}
	gin.SetMode(gin.TestMode)

	radioRecorder := httptest.NewRecorder()
	radioContext, _ := gin.CreateTestContext(radioRecorder)
	radioContext.Request = httptest.NewRequest(http.MethodGet, "/radio", nil)
	GetRadio(radioContext)
	if radioRecorder.Code != http.StatusOK || radioRecorder.Body.Len() == 0 {
		t.Fatalf("radio response status=%d body=%q", radioRecorder.Code, radioRecorder.Body.String())
	}

	videoRecorder := httptest.NewRecorder()
	videoContext, _ := gin.CreateTestContext(videoRecorder)
	videoContext.Request = httptest.NewRequest(http.MethodGet, "/videos", nil)
	GetVideos(videoContext)
	if videoRecorder.Header().Get("Content-Security-Policy") != "default-src 'none'; frame-src 'self' https://video.example; object-src 'none'; base-uri 'none'" {
		t.Fatalf("video CSP=%q", videoRecorder.Header().Get("Content-Security-Policy"))
	}

	agentFeatureFlags.Radio = false
	disabledRecorder := httptest.NewRecorder()
	disabledContext, _ := gin.CreateTestContext(disabledRecorder)
	disabledContext.Request = httptest.NewRequest(http.MethodGet, "/radio", nil)
	GetRadio(disabledContext)
	if disabledRecorder.Body.Len() == 0 || disabledRecorder.Code != http.StatusOK {
		t.Fatalf("disabled radio response status=%d body=%q", disabledRecorder.Code, disabledRecorder.Body.String())
	}
}

func TestVideoAdminControllersDoNotCallServiceWhenFeatureIsDisabled(t *testing.T) {
	previousFlags := agentFeatureFlags
	previousService := videoService
	t.Cleanup(func() {
		agentFeatureFlags = previousFlags
		videoService = previousService
	})
	called := false
	agentFeatureFlags = port.AgentFeatureFlags{}
	videoService = apiVideoServiceFake{called: &called}
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		call   func(*gin.Context)
	}{
		{name: "upload", method: http.MethodPost, path: "/admin/videos/upload", call: UploadVideo},
		{name: "external", method: http.MethodPost, path: "/admin/videos/external", body: `{}`, call: CreateExternalVideo},
		{name: "delete", method: http.MethodDelete, path: "/admin/videos/video-1", call: DeleteVideo},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called = false
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			test.call(context)
			if called {
				t.Fatalf("disabled video controller called the application service")
			}
			if !strings.Contains(recorder.Body.String(), "视频暂未开放") {
				t.Fatalf("disabled video response = %q", recorder.Body.String())
			}
		})
	}
}
