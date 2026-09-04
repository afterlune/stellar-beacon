package middlewares

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SpaceCompanionReadAuth authenticates the read-only internal protocol with a
// token that is separate from the publication token.
func SpaceCompanionReadAuth() gin.HandlerFunc {
	return NewSpaceCompanionAuth(config.SpaceCompanion(), port.SpaceScopeRead, spacePrincipalRepository)
}

// SpaceCompanionPublishAuth authenticates the append-only publication route.
func SpaceCompanionPublishAuth() gin.HandlerFunc {
	return NewSpaceCompanionAuth(config.SpaceCompanion(), port.SpaceScopePublish, spacePrincipalRepository)
}

// NewSpaceCompanionAuth is explicit for unit tests and for future isolated
// deployments. It never logs or copies the presented token into context.
func NewSpaceCompanionAuth(settings config.SpaceCompanionSettings, scope string, principals port.SpacePrincipalRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || !settings.Enabled {
			if c != nil {
				c.AbortWithStatus(http.StatusNotFound)
			}
			return
		}
		if scope == port.SpaceScopePublish && !settings.PublishEnabled {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		expected := settings.ReadToken
		if scope == port.SpaceScopePublish {
			expected = settings.PublishToken
		}
		provided := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.Fields(provided)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || !tokenEqual(parts[1], expected) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if principals == nil {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		record, err := principals.Get(c.Request.Context(), settings.AgentID)
		if err != nil {
			if apperrors.KindOf(err) == apperrors.KindNotFound {
				c.AbortWithStatus(http.StatusUnauthorized)
			} else {
				c.AbortWithStatus(http.StatusServiceUnavailable)
			}
			return
		}
		if !record.Enabled || record.Principal.ID != settings.AgentID || record.Principal.ValidateFor(scope) != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set(port.SpacePrincipalContextKey, record.Principal)
		c.Next()
	}
}

var spacePrincipalRepository port.SpacePrincipalRepository

// ConfigureSpacePrincipalRepository binds the database-backed machine
// principal store during bootstrap. The internal protocol fails closed when
// this dependency is absent, even if a token is configured.
func ConfigureSpacePrincipalRepository(repository port.SpacePrincipalRepository) {
	spacePrincipalRepository = repository
}

func tokenEqual(provided, expected string) bool {
	provided = strings.TrimSpace(provided)
	expected = strings.TrimSpace(expected)
	if provided == "" || expected == "" {
		return false
	}
	providedDigest := sha256.Sum256([]byte(provided))
	expectedDigest := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(providedDigest[:], expectedDigest[:]) == 1
}
