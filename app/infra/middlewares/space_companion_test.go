package middlewares

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeSpacePrincipalRepository struct {
	record port.SpacePrincipalRecord
	err    error
}

func (f fakeSpacePrincipalRepository) Get(context.Context, string) (port.SpacePrincipalRecord, error) {
	return f.record, f.err
}

func TestSpaceCompanionAuthUsesSeparateConstantTimeScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := config.SpaceCompanionSettings{
		Enabled:        true,
		PublishEnabled: true,
		AgentID:        port.MoonfeiPrincipalID,
		ReadToken:      "read-token-for-space-companion-0123456789",
		PublishToken:   "publish-token-for-space-companion-0123456789",
	}
	principals := fakeSpacePrincipalRepository{record: port.SpacePrincipalRecord{
		Principal: port.SpacePrincipal{
			ID:     port.MoonfeiPrincipalID,
			Type:   port.SpacePrincipalAgent,
			Scopes: []string{port.SpaceScopeRead, port.SpaceScopePublish},
		},
		Enabled: true,
	}}

	router := gin.New()
	router.GET("/read", NewSpaceCompanionAuth(settings, port.SpaceScopeRead, principals), func(c *gin.Context) {
		principal, ok := c.Get(port.SpacePrincipalContextKey)
		if !ok || principal.(port.SpacePrincipal).ID != port.MoonfeiPrincipalID {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})
	router.POST("/publish", NewSpaceCompanionAuth(settings, port.SpaceScopePublish, principals), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	cases := []struct {
		name   string
		method string
		path   string
		token  string
		want   int
	}{
		{"read token reads", http.MethodGet, "/read", settings.ReadToken, http.StatusNoContent},
		{"publish token publishes", http.MethodPost, "/publish", settings.PublishToken, http.StatusNoContent},
		{"publish token cannot read", http.MethodGet, "/read", settings.PublishToken, http.StatusUnauthorized},
		{"malformed bearer is rejected", http.MethodGet, "/read", "Bearer one two", http.StatusUnauthorized},
		{"wrong token is rejected", http.MethodGet, "/read", "not-the-token", http.StatusUnauthorized},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			if testCase.token != "" {
				request.Header.Set("Authorization", "Bearer "+testCase.token)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != testCase.want {
				t.Fatalf("status = %d, want %d", response.Code, testCase.want)
			}
		})
	}
}

func TestSpaceCompanionAuthFailsClosedWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/read", NewSpaceCompanionAuth(config.SpaceCompanionSettings{}, port.SpaceScopeRead, nil), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/read", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestSpaceCompanionAuthRequiresEnabledDatabasePrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := config.SpaceCompanionSettings{
		Enabled:   true,
		AgentID:   port.MoonfeiPrincipalID,
		ReadToken: "read-token-for-space-companion-0123456789",
	}
	router := gin.New()
	router.GET("/read", NewSpaceCompanionAuth(settings, port.SpaceScopeRead, fakeSpacePrincipalRepository{
		record: port.SpacePrincipalRecord{Principal: port.SpacePrincipal{ID: port.MoonfeiPrincipalID, Type: port.SpacePrincipalAgent}, Enabled: false},
	}), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/read", nil)
	request.Header.Set("Authorization", "Bearer "+settings.ReadToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("disabled principal status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	router = gin.New()
	router.GET("/read", NewSpaceCompanionAuth(settings, port.SpaceScopeRead, nil), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing principal store status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
