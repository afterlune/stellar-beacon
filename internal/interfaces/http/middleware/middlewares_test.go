package middlewares

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// An aborted request must not be filed as an exception: the caller is gone, so
// the failure is a consequence of the disconnect rather than a service fault.
// Public routes must attach the logged-in reader, otherwise comment writes and
// private-article access can never see an account.
func TestAttachOptionalLoginUserIgnoresUnusableTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, header := range []string{"", "Bearer", "Bearer null", "Basic abc", "Bearer not-a-jwt"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/public/comments", nil)
		if header != "" {
			c.Request.Header.Set("Authorization", header)
		}
		attachOptionalLoginUser(c)
		if _, ok := c.Get("userInfo"); ok {
			t.Fatalf("header %q must not attach an account", header)
		}
	}
}

func TestShouldRecordExceptionSkipsCanceledRequests(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		ctx  context.Context
		code string
		want bool
	}{
		{name: "application failure", ctx: context.Background(), code: "OPERATION_FAILED", want: true},
		{name: "invalid argument", ctx: context.Background(), code: "INVALID_ARGUMENT", want: true},
		{name: "success", ctx: context.Background(), code: "OK", want: false},
		{name: "canceled request", ctx: canceled, code: "OPERATION_FAILED", want: false},
		{name: "nil context", ctx: nil, code: "OPERATION_FAILED", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRecordException(tt.ctx, tt.code); got != tt.want {
				t.Fatalf("shouldRecordException(%v, %q) = %v, want %v", tt.ctx, tt.code, got, tt.want)
			}
		})
	}
}

func TestIsAdminPathUsesSegmentBoundary(t *testing.T) {
	tests := []struct {
		path  string
		admin bool
	}{
		{path: "/v1/admin", admin: true},
		{path: "/v1/admin/roles", admin: true},
		{path: "/admin", admin: false},
		{path: "/v1/administrator", admin: false},
		{path: "/v1/administer", admin: false},
	}
	for _, tt := range tests {
		if got := isAdminPath(tt.path); got != tt.admin {
			t.Errorf("isAdminPath(%q) = %v, want %v", tt.path, got, tt.admin)
		}
	}
}

func TestRequiresAuthentication(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/v1/admin/articles", want: true},
		{path: "/v1/auth/logout", want: true},
		{path: "/v1/auth/me", want: true},
		{path: "/v1/auth/me/avatar", want: true},
		{path: "/v1/auth/me/reactions", want: true},
		{path: "/v1/auth/me/reactions/state", want: true},
		{path: "/v1/auth/me/collection-reactions", want: true},
		{path: "/v1/auth/me/collection-reactions/state", want: true},
		{path: "/v1/auth/me/comment-reactions", want: true},
		{path: "/v1/auth/login", want: false},
		{path: "/v1/auth/password", want: false},
		{path: "/v1/public/articles", want: false},
	}
	for _, tt := range tests {
		if got := requiresAuthentication(tt.path); got != tt.want {
			t.Errorf("requiresAuthentication(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestSwaggerPathMatchesParameters(t *testing.T) {
	tests := []struct {
		template string
		request  string
		want     bool
	}{
		{template: "/v1/admin/albums/{albumId}", request: "/v1/admin/albums/21", want: true},
		{template: "/v1/admin/albums/{albumId}/photos", request: "/v1/admin/albums/21/photos", want: true},
		{template: "/v1/admin/albums/{albumId}", request: "/v1/admin/albums/21/photos", want: false},
		{template: "/v1/admin/albums/{albumId}", request: "/v1/admin/albums/", want: false},
		{template: "/v1/admin/albums", request: "/v1/admin/albums/21", want: false},
	}
	for _, tt := range tests {
		if got := swaggerPathMatches(tt.template, tt.request); got != tt.want {
			t.Errorf("swaggerPathMatches(%q, %q) = %v, want %v", tt.template, tt.request, got, tt.want)
		}
	}
}

func TestRequestLogPayloadOmitsMultipartBytes(t *testing.T) {
	body := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=test")

	got := requestLogPayload(req, body)
	if !utf8.ValidString(got) {
		t.Fatal("request log payload must be valid UTF-8")
	}
	if !strings.Contains(got, "request body omitted") || !strings.Contains(got, "multipart/form-data") {
		t.Fatalf("unexpected sanitized payload: %q", got)
	}
}

func TestTruncateLogTextPreservesUTF8(t *testing.T) {
	got := truncateLogText(strings.Repeat("中", 1000), 2000)
	if !utf8.ValidString(got) {
		t.Fatal("truncated log payload must be valid UTF-8")
	}
	if len(got) > 2000 || !strings.HasSuffix(got, "...[truncated]") {
		t.Fatalf("unexpected truncated payload length or suffix: %d %q", len(got), got[len(got)-minInt(len(got), 20):])
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func TestIsReadLimitedRequestKeepsBodyRecommendationOnReadBudget(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{method: http.MethodGet, path: "/v1/public/feed", want: true},
		{method: http.MethodGet, path: "/v1/auth/me/notifications/unread-count", want: true},
		{method: http.MethodGet, path: "/v1/auth/verification-code", want: false},
		{method: http.MethodPost, path: "/v1/auth/me/recommendations/query", want: true},
		{method: http.MethodPost, path: "/v1/auth/me/topic-subscriptions/tag/go", want: false},
	}
	for _, tt := range tests {
		if got := isReadLimitedRequest(tt.method, tt.path); got != tt.want {
			t.Fatalf("isReadLimitedRequest(%s, %s) = %v, want %v", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestRecommendationQueryOmitsSeedsFromOperationAndExceptionLogs(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/me/recommendations/query", strings.NewReader(`{"seedArticleIds":[42]}`))
	payload := requestLogPayload(req, []byte(`{"seedArticleIds":[42]}`))
	if strings.Contains(payload, "42") {
		t.Fatalf("recommendation seeds leaked into log payload: %q", payload)
	}
	if shouldRecordOperation(http.MethodPost, "/v1/auth/me/recommendations/query") {
		t.Fatal("read-only recommendation queries must not create operation-log rows")
	}
}

func TestSwaggerPathDataPrefersStaticSegments(t *testing.T) {
	apis := map[string]interface{}{
		"/v1/studio/collections/{collectionId}/comments/{commentId}": map[string]interface{}{
			"delete": map[string]interface{}{"summary": "param"},
		},
		"/v1/studio/collections/{collectionId}/comments/batch": map[string]interface{}{
			"post": map[string]interface{}{"summary": "static"},
		},
	}
	for attempt := 0; attempt < 50; attempt++ {
		data, ok := swaggerPathData(apis, "/v1/studio/collections/7/comments/batch")
		if !ok {
			t.Fatal("batch path must resolve")
		}
		if _, ok := data["post"]; !ok {
			t.Fatalf("static batch path must win over the parameterised sibling: %+v", data)
		}
	}
}
