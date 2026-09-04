package service

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type agentProfileRepositoryFake struct {
	profile port.AgentProfile
	saved   port.AgentProfile
	err     error
}

func (f *agentProfileRepositoryFake) Get(context.Context, string) (port.AgentProfile, error) {
	if f.err != nil {
		return port.AgentProfile{}, f.err
	}
	return f.profile, nil
}

func (f *agentProfileRepositoryFake) Save(_ context.Context, profile port.AgentProfile) error {
	f.saved = profile
	return f.err
}

func profileTestContext(method string, body any) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	payload, _ := json.Marshal(body)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, "/admin/ai/profile", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	return serviceTestRequest{ginContextForServiceTest: c}
}

func TestAgentProfileServiceReturnsAndPatchesVersionedPersona(t *testing.T) {
	repo := &agentProfileRepositoryFake{profile: port.AgentProfile{
		ID:            "benetnasch-public",
		Name:          "Benetnasch",
		PromptVersion: "v1",
		SystemPrompt:  "保持边界",
		Opening:       "你好",
		RhythmPrompts: map[port.AgentRhythmPhase]string{port.AgentRhythmAwake: "清醒", port.AgentRhythmDusk: "黄昏", port.AgentRhythmNight: "深夜"},
		Enabled:       true,
	}}
	service := NewAgentProfileService(repo, "benetnasch-public")
	getContext := profileTestContext(http.MethodGet, nil)
	result := service.Get(getContext)
	if !result.Flag {
		t.Fatalf("Get() failed: %+v", result)
	}
	dto, ok := result.Data.(model.AgentProfileDTO)
	if !ok || dto.ID != "benetnasch-public" || dto.RhythmPrompts["night"] != "深夜" {
		t.Fatalf("Get() data = %#v", result.Data)
	}

	result = service.Update(profileTestContext(http.MethodPatch, map[string]any{
		"name":          "新 Benetnasch",
		"promptVersion": "v2",
		"systemPrompt":  "新的系统边界",
		"awakePrompt":   "新的清醒提示",
		"enabled":       false,
	}))
	if !result.Flag {
		t.Fatalf("Update() failed: %+v", result)
	}
	if repo.saved.Name != "新 Benetnasch" || repo.saved.PromptVersion != "v2" || repo.saved.RhythmPrompts[port.AgentRhythmAwake] != "新的清醒提示" || repo.saved.RhythmPrompts[port.AgentRhythmNight] != "深夜" || repo.saved.Enabled {
		t.Fatalf("saved profile = %#v", repo.saved)
	}
}

func TestAgentProfileServiceRejectsOversizedOrInvalidPersona(t *testing.T) {
	repo := &agentProfileRepositoryFake{profile: port.AgentProfile{ID: "profile", Name: "name", PromptVersion: "v1", SystemPrompt: "prompt", Enabled: true}}
	service := NewAgentProfileService(repo, "profile")
	result := service.Update(profileTestContext(http.MethodPatch, map[string]any{"systemPrompt": strings.Repeat("x", 8001)}))
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("oversized profile result = %+v", result)
	}
	if repo.saved.ID != "" {
		t.Fatal("invalid profile must not be saved")
	}
}

func TestAgentProfileServiceSurfacesRepositoryErrors(t *testing.T) {
	repo := &agentProfileRepositoryFake{
		profile: port.AgentProfile{ID: "profile", Name: "name", PromptVersion: "v1", SystemPrompt: "prompt"},
		err:     apperrors.Unavailable("agent.profile.save", errors.New("secret backend detail")),
	}
	service := NewAgentProfileService(repo, "profile")
	result := service.Update(profileTestContext(http.MethodPatch, map[string]any{"name": "updated"}))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("repository error result = %+v", result)
	}
}
