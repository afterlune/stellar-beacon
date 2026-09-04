package middlewares

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestIsAdminPathUsesSegmentBoundary(t *testing.T) {
	tests := []struct {
		path  string
		admin bool
	}{
		{path: "/admin", admin: true},
		{path: "/admin/roles", admin: true},
		{path: "/administrator", admin: false},
		{path: "/administer", admin: false},
	}
	for _, tt := range tests {
		if got := isAdminPath(tt.path); got != tt.admin {
			t.Errorf("isAdminPath(%q) = %v, want %v", tt.path, got, tt.admin)
		}
	}
}

func TestIsTimeCapsulePathUsesSegmentBoundary(t *testing.T) {
	for _, test := range []struct {
		path string
		want bool
	}{
		{path: "/capsules", want: true},
		{path: "/capsules/capsule-1", want: true},
		{path: "/capsulesque", want: false},
		{path: "/public/capsules", want: false},
	} {
		if got := isTimeCapsulePath(test.path); got != test.want {
			t.Errorf("isTimeCapsulePath(%q) = %v, want %v", test.path, got, test.want)
		}
	}
}

func TestGetSwaggerInfoMatchesDynamicPathSegments(t *testing.T) {
	previous := swaggerCache
	t.Cleanup(func() { swaggerCache = previous })
	swaggerCache = map[string]interface{}{
		"paths": map[string]interface{}{
			"/admin/items/:id": map[string]interface{}{
				"delete": map[string]interface{}{
					"summary":     "项目",
					"description": "删除项目",
				},
			},
		},
	}

	module, description := getSwaggerInfo("/admin/items/42", http.MethodDelete)
	if module != "项目" || description != "删除项目" {
		t.Fatalf("getSwaggerInfo() = %q, %q; want %q, %q", module, description, "项目", "删除项目")
	}
}

func TestGetSwaggerInfoReturnsUnknownForUnlistedRouteWithoutError(t *testing.T) {
	previous := swaggerCache
	t.Cleanup(func() { swaggerCache = previous })
	swaggerCache = map[string]interface{}{
		"paths": map[string]interface{}{},
	}

	module, description := getSwaggerInfo("/admin/not-in-swagger", http.MethodGet)
	if module != "Unknown" || description != "Unknown" {
		t.Fatalf("getSwaggerInfo() = %q, %q; want Unknown metadata", module, description)
	}
}

func TestProtectedUserPathRequiresIdentityOnlyForMutations(t *testing.T) {
	for _, test := range []struct {
		path   string
		method string
		want   bool
	}{
		{path: "/users/info", method: http.MethodPut, want: true},
		{path: "/users/avatar", method: http.MethodPost, want: true},
		{path: "/users/email", method: http.MethodPut, want: true},
		{path: "/users/subscribe", method: http.MethodPut, want: true},
		{path: "/users/info/7", method: http.MethodGet, want: false},
		{path: "/users/password", method: http.MethodPut, want: false},
		{path: "/users/register", method: http.MethodPost, want: false},
	} {
		if got := isProtectedUserPath(test.path, test.method); got != test.want {
			t.Errorf("isProtectedUserPath(%q, %q) = %v, want %v", test.path, test.method, got, test.want)
		}
	}
}

func TestUserDetailsFromContextRejectsInvalidIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name  string
		value any
		set   bool
	}{
		{name: "missing"},
		{name: "wrong type", value: "user", set: true},
		{name: "zero id", value: model.UserDetailsDTO{UserInfoId: 0}, set: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if test.set {
				c.Set("userInfo", test.value)
			}
			if _, ok := userDetailsFromContext(c); ok {
				t.Fatal("invalid identity was accepted")
			}
		})
	}
}

func TestUserDetailsFromContextAcceptsAuthenticatedIdentity(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	want := model.UserDetailsDTO{UserInfoId: 7, Nickname: "admin"}
	c.Set("userInfo", want)
	got, ok := userDetailsFromContext(c)
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("userDetailsFromContext() = %#v, %v; want %#v, true", got, ok, want)
	}
}

func TestCasbinResourceFilterRejectsInvalidIdentityWithoutPanic(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/user/menus", nil)
	c.Set("userInfo", "not-a-user-details-dto")

	CasbinResourceFilter()(c)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestGetRolesByUserInfoIDPropagatesRepositoryError(t *testing.T) {
	previousRepo, previousCache := roleRepo, roleCache
	t.Cleanup(func() {
		roleRepo = previousRepo
		roleCache = previousCache
	})
	roleCache = nil
	wantErr := errors.New("role database unavailable")
	roleRepo = &middlewareRoleRepository{err: wantErr}

	_, err := getRolesByUserInfoId(context.Background(), 7)
	if !errors.Is(err, wantErr) {
		t.Fatalf("getRolesByUserInfoId() error = %v, want %v", err, wantErr)
	}
}

func TestGetRolesByUserInfoIDFallsBackToRepositoryWithoutCache(t *testing.T) {
	previousRepo, previousCache := roleRepo, roleCache
	t.Cleanup(func() {
		roleRepo = previousRepo
		roleCache = previousCache
	})
	roleCache = nil
	roleRepo = &middlewareRoleRepository{roles: []string{"admin", "editor"}}

	roles, err := getRolesByUserInfoId(context.Background(), 7)
	if err != nil {
		t.Fatalf("getRolesByUserInfoId() error = %v", err)
	}
	if !reflect.DeepEqual(roles, []string{"admin", "editor"}) {
		t.Fatalf("roles = %#v, want %#v", roles, []string{"admin", "editor"})
	}
}

func TestCasbinResourceFilterReturnsUnavailableWhenRolesCannotLoad(t *testing.T) {
	previousRepo, previousCache := roleRepo, roleCache
	t.Cleanup(func() {
		roleRepo = previousRepo
		roleCache = previousCache
	})
	roleCache = nil
	roleRepo = &middlewareRoleRepository{err: errors.New("role database unavailable")}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/user/menus", nil)
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})

	CasbinResourceFilter()(c)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestResourcePermissionGrantedMatchesConfiguredRoleAndPath(t *testing.T) {
	resources := []port.ResourceRoleView{
		{Url: "/admin/ai/profile", RequestMethod: "PATCH", RoleList: []string{"admin"}},
		{Url: "/admin/ai/reviews/*/approve", RequestMethod: "POST", RoleList: []string{"editor"}},
	}

	if !resourcePermissionGranted([]string{"admin"}, resources, "/admin/ai/profile", http.MethodPatch) {
		t.Fatal("configured role should be allowed to update the Agent profile")
	}
	if !resourcePermissionGranted([]string{"editor"}, resources, "/admin/ai/reviews/42/approve", http.MethodPost) {
		t.Fatal("configured role should be allowed to approve the matching review")
	}
}

func TestResourcePermissionGrantedRejectsRoleMethodAndPathMismatches(t *testing.T) {
	resources := []port.ResourceRoleView{
		{Url: "/admin/ai/profile", RequestMethod: "PATCH", RoleList: []string{"admin"}},
		{Url: "/admin/ai/reviews/*/approve", RequestMethod: "POST", RoleList: []string{"editor"}},
	}

	tests := []struct {
		name   string
		roles  []string
		uri    string
		method string
	}{
		{name: "wrong role", roles: []string{"user"}, uri: "/admin/ai/profile", method: http.MethodPatch},
		{name: "wrong method", roles: []string{"admin"}, uri: "/admin/ai/profile", method: http.MethodPost},
		{name: "wrong path", roles: []string{"admin"}, uri: "/admin/ai/profile/42", method: http.MethodPatch},
		{name: "wrong review action", roles: []string{"editor"}, uri: "/admin/ai/reviews/42/reject", method: http.MethodPost},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if resourcePermissionGranted(test.roles, resources, test.uri, test.method) {
				t.Fatalf("resourcePermissionGranted(%v, %q, %q) unexpectedly allowed", test.roles, test.uri, test.method)
			}
		})
	}
}

func TestAccessLimiterDoesNotBypassAuthenticatedRequests(t *testing.T) {
	previousLimiters, previousGlobal := ipLimiter, globalLimiter
	t.Cleanup(func() {
		mutex.Lock()
		ipLimiter = previousLimiters
		globalLimiter = previousGlobal
		mutex.Unlock()
	})

	clientIP := "198.51.100.9"
	mutex.Lock()
	ipLimiter = map[string]*rate.Limiter{
		clientIP: rate.NewLimiter(rate.Limit(0), 0),
	}
	globalLimiter = rate.NewLimiter(rate.Inf, 1)
	mutex.Unlock()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/user/menus", nil)
	c.Request.RemoteAddr = net.JoinHostPort(clientIP, "443")
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})

	AccessLimiter()(c)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d for a rate-limited authenticated request", recorder.Code, http.StatusTooManyRequests)
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

func TestCaptureRequestBodySkipsMultipartWithoutConsumingIt(t *testing.T) {
	want := bytes.Repeat([]byte("video"), requestBodyCaptureLimit)
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(want))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=test")

	capture, err := captureRequestBody(req)
	if err != nil {
		t.Fatalf("captureRequestBody() error = %v", err)
	}
	if !capture.omitted || len(capture.data) != 0 {
		t.Fatalf("multipart capture = %#v, want omitted without data", capture)
	}
	got, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read replayed multipart body: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("replayed multipart body length = %d, want %d", len(got), len(want))
	}
}

func TestCaptureRequestBodyBoundsUnknownLengthAndReplaysFullBody(t *testing.T) {
	want := bytes.Repeat([]byte("x"), requestBodyCaptureLimit+1024)
	req := httptest.NewRequest(http.MethodPost, "/large", bytes.NewReader(want))
	req.ContentLength = -1

	capture, err := captureRequestBody(req)
	if err != nil {
		t.Fatalf("captureRequestBody() error = %v", err)
	}
	if !capture.omitted || len(capture.data) != 0 {
		t.Fatalf("large capture = %#v, want omitted without data", capture)
	}
	got, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read replayed large body: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("replayed large body length = %d, want %d", len(got), len(want))
	}
}

func TestCaptureRequestBodyCapturesSmallBodyAndReplaysIt(t *testing.T) {
	want := []byte(`{"title":"small"}`)
	req := httptest.NewRequest(http.MethodPost, "/small", bytes.NewReader(want))

	capture, err := captureRequestBody(req)
	if err != nil {
		t.Fatalf("captureRequestBody() error = %v", err)
	}
	if capture.omitted || !bytes.Equal(capture.data, want) {
		t.Fatalf("small capture = %#v, want captured body", capture)
	}
	got, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read replayed small body: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("replayed small body = %q, want %q", got, want)
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

func TestResponseCodeFromBodyRequiresValidJSONEnvelope(t *testing.T) {
	for _, test := range []struct {
		name      string
		body      string
		wantCode  int
		wantValid bool
	}{
		{name: "valid error response", body: `{"code":51000,"message":"failed"}`, wantCode: 51000, wantValid: true},
		{name: "valid success response", body: `{"code":20000}`, wantCode: 20000, wantValid: true},
		{name: "malformed response", body: `{"code":51000`, wantValid: false},
		{name: "empty response", body: "", wantValid: false},
		{name: "string code", body: `{"code":"51000"}`, wantValid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotCode, gotValid := responseCodeFromBody([]byte(test.body))
			if gotCode != test.wantCode || gotValid != test.wantValid {
				t.Fatalf("responseCodeFromBody(%q) = %d, %v; want %d, %v", test.body, gotCode, gotValid, test.wantCode, test.wantValid)
			}
		})
	}
}

func TestProjectUserInfoDTOSelectsOnlyPublicLoginFields(t *testing.T) {
	details := model.UserDetailsDTO{
		Id: 7, UserInfoId: 8, Email: "user@example.com", Username: "user@example.com",
		Password: "secret", Roles: []string{"admin"}, Nickname: "用户", IpAddress: "127.0.0.1",
	}
	got := projectUserInfoDTO(details, "access-token")
	if got.Id != 7 || got.UserInfoId != 8 || got.Username != details.Username || got.Token != "access-token" {
		t.Fatalf("unexpected projected login DTO: %+v", got)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal projected login DTO: %v", err)
	}
	if strings.Contains(string(payload), "secret") || strings.Contains(string(payload), "admin") || strings.Contains(string(payload), "password") || strings.Contains(string(payload), "roles") {
		t.Fatalf("private login fields leaked: %s", payload)
	}
}

type middlewareRoleRepository struct {
	roles []string
	err   error
}

func (f *middlewareRoleRepository) ListUserRoles(context.Context) ([]entity.TRole, error) {
	return nil, f.err
}

func (f *middlewareRoleRepository) Count(context.Context, string) (int64, error) {
	return 0, f.err
}

func (f *middlewareRoleRepository) List(context.Context, int, int, string) ([]port.RoleView, error) {
	return nil, f.err
}

func (f *middlewareRoleRepository) FindByName(context.Context, string) (entity.TRole, error) {
	return entity.TRole{}, f.err
}

func (f *middlewareRoleRepository) SaveOrUpdate(context.Context, entity.TRole, []int, []int) error {
	return f.err
}

func (f *middlewareRoleRepository) Delete(context.Context, []int) error {
	return f.err
}

func (f *middlewareRoleRepository) ListResourceRoles(context.Context) ([]port.ResourceRoleView, error) {
	return nil, f.err
}

func (f *middlewareRoleRepository) ListRolesByUserInfoID(context.Context, int) ([]string, error) {
	return f.roles, f.err
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
