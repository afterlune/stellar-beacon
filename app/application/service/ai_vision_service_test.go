package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"

	"github.com/gin-gonic/gin"
)

type visionChatGatewayFake struct {
	response port.ChatResponse
	err      error
	request  port.ChatRequest
	calls    int
}

func (f *visionChatGatewayFake) Generate(_ context.Context, request port.ChatRequest) (port.ChatResponse, error) {
	f.request = request
	f.calls++
	return f.response, f.err
}

func (f *visionChatGatewayFake) Stream(context.Context, port.ChatRequest, func(port.ChatStreamEvent) error) error {
	return nil
}

func visionServiceContext(method, body string) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, "/admin/ai/vision/preview", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	return serviceTestRequest{ginContextForServiceTest: c}
}

func newVisionServiceForTest(t *testing.T, gateway port.ChatGateway, reviews *fakeAIReviewRepository) *MyAIVisionService {
	t.Helper()
	service, err := NewAIVisionService(AIVisionServiceDeps{
		Vision:          gateway,
		Reviews:         reviews,
		ReviewPolicy:    nil,
		ReviewPolicyID:  "default",
		Model:           "vision-model",
		MaxOutputTokens: 123,
		Timeout:         time.Second,
		Enabled:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := service.(*MyAIVisionService)
	if !ok {
		t.Fatalf("service type = %T", service)
	}
	return result
}

func TestAIVisionPreviewCreatesPendingReviewAndUsesVisionUseCase(t *testing.T) {
	image := base64.StdEncoding.EncodeToString([]byte("small-image"))
	gateway := &visionChatGatewayFake{response: port.ChatResponse{RunID: "vision-run-1", Text: "图片中有一棵树。"}}
	reviews := &fakeAIReviewRepository{}
	service := newVisionServiceForTest(t, gateway, reviews)

	result := service.Preview(visionServiceContext(http.MethodPost, `{"prompt":"请描述图片","imageBase64":"`+image+`","mimeType":"image/png","detail":"low"}`))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	preview, ok := result.Data.(model.AIVisionPreviewDTO)
	if !ok || preview.ReviewID == "" || preview.RunID != "vision-run-1" || preview.Operation != "vision" || preview.Preview != "图片中有一棵树。" {
		t.Fatalf("preview = %#v", result.Data)
	}
	if gateway.calls != 1 || gateway.request.UseCase != port.AIUseCaseVision || gateway.request.Model != "vision-model" || len(gateway.request.Messages) != 2 {
		t.Fatalf("vision request = %#v calls=%d", gateway.request, gateway.calls)
	}
	if gateway.request.Messages[0].Role != port.ChatRoleSystem || len(gateway.request.Messages[1].ContentParts) != 1 || gateway.request.Messages[1].ContentParts[0].Base64Data != image {
		t.Fatalf("vision message = %#v", gateway.request.Messages)
	}
	if _, exists := reviews.reviews[preview.ReviewID]; !exists || reviews.reviews[preview.ReviewID].Status != port.ReviewPending {
		t.Fatalf("pending review = %#v", reviews.reviews)
	}
}

func TestAIVisionPreviewRejectsUnsafeOrAmbiguousImageBeforeProviderCall(t *testing.T) {
	gateway := &visionChatGatewayFake{response: port.ChatResponse{Text: "should not run"}}
	service := newVisionServiceForTest(t, gateway, &fakeAIReviewRepository{})
	image := base64.StdEncoding.EncodeToString([]byte("image"))
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "both sources", body: `{"prompt":"describe","imageUrl":"https://example.com/a.png","imageBase64":"` + image + `","mimeType":"image/png"}`},
		{name: "no source", body: `{"prompt":"describe"}`},
		{name: "private URL", body: `{"prompt":"describe","imageUrl":"http://127.0.0.1/image.png"}`},
		{name: "invalid MIME", body: `{"prompt":"describe","imageBase64":"` + image + `","mimeType":"image/svg+xml"}`},
		{name: "unknown field", body: `{"prompt":"describe","imageUrl":"https://example.com/a.png","unexpected":true}`},
		{name: "control prompt", body: "{\"prompt\":\"describe\\u0000\",\"imageUrl\":\"https://example.com/a.png\"}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := service.Preview(visionServiceContext(http.MethodPost, test.body))
			if result.Flag || result.Message != "参数格式不正确" {
				t.Fatalf("result = %+v", result)
			}
		})
	}
	if gateway.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", gateway.calls)
	}
}

func TestAIVisionPreviewFailsClosedWhenDisabled(t *testing.T) {
	gateway := &visionChatGatewayFake{response: port.ChatResponse{Text: "should not run"}}
	service := NewDisabledAIVisionService()
	result := service.Preview(visionServiceContext(http.MethodPost, `{"prompt":"describe","imageUrl":"https://example.com/a.png"}`))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" || gateway.calls != 0 {
		t.Fatalf("disabled result = %+v calls=%d", result, gateway.calls)
	}
}

func TestAIVisionPreviewDoesNotPersistInvalidProviderOutput(t *testing.T) {
	secretErr := apperrors.Unavailable("provider.request", errors.New("api_key=secret response=private"))
	gateway := &visionChatGatewayFake{err: secretErr}
	reviews := &fakeAIReviewRepository{}
	service := newVisionServiceForTest(t, gateway, reviews)
	result := service.Preview(visionServiceContext(http.MethodPost, `{"prompt":"describe","imageUrl":"https://example.com/a.png"}`))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" || len(reviews.reviews) != 0 || strings.Contains(result.Message, "secret") {
		t.Fatalf("provider failure result = %+v reviews=%#v", result, reviews.reviews)
	}

	gateway.err = nil
	gateway.response = port.ChatResponse{Text: ""}
	result = service.Preview(visionServiceContext(http.MethodPost, `{"prompt":"describe","imageUrl":"https://example.com/a.png"}`))
	if result.Flag || len(reviews.reviews) != 0 {
		t.Fatalf("empty output result = %+v reviews=%#v", result, reviews.reviews)
	}
}

func TestAIVisionPreviewRejectsOversizedBody(t *testing.T) {
	gateway := &visionChatGatewayFake{}
	service := newVisionServiceForTest(t, gateway, &fakeAIReviewRepository{})
	body := `{"prompt":"describe","imageUrl":"https://example.com/a.png","padding":"` + strings.Repeat("x", int(visionPreviewBodyLimit)) + `"}`
	result := service.Preview(visionServiceContext(http.MethodPost, body))
	if result.Flag || result.Message != "参数格式不正确" || gateway.calls != 0 {
		t.Fatalf("oversized result = %+v calls=%d", result, gateway.calls)
	}
	if !bytes.Contains([]byte(body), []byte("padding")) {
		t.Fatal("test body was not constructed")
	}
}
