package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/gin-gonic/gin"
)

type sessionRevocationUserRepository struct {
	port.UserInfoRepository
	auth        entity.TUserAuth
	updateError error
	findError   error
	updatedID   int
	disabled    int
}

func (r *sessionRevocationUserRepository) UpdateDisable(_ context.Context, id, disabled int) error {
	r.updatedID, r.disabled = id, disabled
	return r.updateError
}

func (r *sessionRevocationUserRepository) FindAuthByUserInfoID(_ context.Context, id int) (entity.TUserAuth, error) {
	r.updatedID = id
	return r.auth, r.findError
}

func (r *sessionRevocationUserRepository) IsEnabled(context.Context, int) (bool, error) {
	return false, nil
}

type sessionRevocationCache struct {
	fakeServiceCache
	hashDeletes [][2]string
	deletedKeys []string
	hashError   error
	deleteError error
}

func (c *sessionRevocationCache) HDel(_ context.Context, key, field string) error {
	c.hashDeletes = append(c.hashDeletes, [2]string{key, field})
	return c.hashError
}

func (c *sessionRevocationCache) Delete(_ context.Context, key string) error {
	c.deletedKeys = append(c.deletedKeys, key)
	return c.deleteError
}

func newSessionRevocationService(t *testing.T, repo *sessionRevocationUserRepository, cache *sessionRevocationCache) *MyUserInfoService {
	t.Helper()
	userInfoService, err := NewUserInfoService(UserInfoServiceDeps{
		Repo: repo, Cache: cache, Storage: fakeServiceStorage{},
	})
	if err != nil {
		t.Fatalf("create user info service: %v", err)
	}
	return userInfoService
}

func disableUserContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPut, "/v1/admin/users/disable", strings.NewReader(`{"id":18,"isDisable":1}`))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestUpdateUserDisableRevokesAccessAndRefreshSessions(t *testing.T) {
	repo := &sessionRevocationUserRepository{auth: entity.TUserAuth{Id: 77}}
	cache := &sessionRevocationCache{}
	result := newSessionRevocationService(t, repo, cache).UpdateUserDisable(disableUserContext())

	if !result.Flag {
		t.Fatalf("disable user failed: %+v", result)
	}
	if repo.updatedID != 18 || repo.disabled != 1 {
		t.Fatalf("unexpected user disable input: id=%d disabled=%d", repo.updatedID, repo.disabled)
	}
	if len(cache.hashDeletes) != 1 || cache.hashDeletes[0] != [2]string{LoginUser, "77"} {
		t.Fatalf("unexpected login session revocation: %v", cache.hashDeletes)
	}
	if len(cache.deletedKeys) != 1 || cache.deletedKeys[0] != RefreshTokenPrefix+"77" {
		t.Fatalf("unexpected refresh session revocation: %v", cache.deletedKeys)
	}
}

func TestUpdateUserDisableAttemptsBothRevocationsOnCacheError(t *testing.T) {
	repo := &sessionRevocationUserRepository{auth: entity.TUserAuth{Id: 77}}
	cache := &sessionRevocationCache{hashError: errors.New("hash unavailable"), deleteError: errors.New("redis unavailable")}
	result := newSessionRevocationService(t, repo, cache).UpdateUserDisable(disableUserContext())

	if result.Flag {
		t.Fatalf("cache cleanup failure must be reported: %+v", result)
	}
	if len(cache.hashDeletes) != 1 || len(cache.deletedKeys) != 1 {
		t.Fatalf("both session stores must be cleaned even if one operation fails: hashes=%v keys=%v", cache.hashDeletes, cache.deletedKeys)
	}
}

func TestRemoveOnlineUserRevokesAccessAndRefreshSessions(t *testing.T) {
	repo := &sessionRevocationUserRepository{auth: entity.TUserAuth{Id: 93}}
	cache := &sessionRevocationCache{}
	userInfoService := newSessionRevocationService(t, repo, cache)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodDelete, "/v1/admin/users/18/online", nil)
	c.Params = gin.Params{{Key: "userInfoId", Value: "18"}}

	result := userInfoService.RemoveOnlineUser(c)
	if !result.Flag {
		t.Fatalf("remove online user failed: %+v", result)
	}
	if len(cache.hashDeletes) != 1 || cache.hashDeletes[0] != [2]string{LoginUser, "93"} {
		t.Fatalf("unexpected login session revocation: %v", cache.hashDeletes)
	}
	if len(cache.deletedKeys) != 1 || cache.deletedKeys[0] != RefreshTokenPrefix+"93" {
		t.Fatalf("unexpected refresh session revocation: %v", cache.deletedKeys)
	}
}
