package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type fakeAuthRepository struct {
	user port.AuthUser
	err  error
}

func (f *fakeAuthRepository) ListUsers(context.Context, port.UserFilter) ([]*port.UserAdmin, int64, error) {
	return nil, 0, nil
}
func (f *fakeAuthRepository) FindByUsername(context.Context, string) (port.AuthUser, error) {
	return f.user, f.err
}
func (f *fakeAuthRepository) FindByID(context.Context, int) (entity.TUserAuth, error) {
	return f.user.Auth, f.err
}
func (f *fakeAuthRepository) CreateUser(context.Context, entity.TUserInfo, entity.TUserAuth, int) error {
	return nil
}
func (f *fakeAuthRepository) UpdatePassword(context.Context, string, string) error { return nil }
func (f *fakeAuthRepository) UpdatePasswordByID(context.Context, int, string) error {
	return nil
}
func (f *fakeAuthRepository) UpdateLoginMetadata(context.Context, entity.TUserAuth) error {
	return nil
}

func TestUserAuthServiceAuthenticatesThroughPort(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	service := mustUserAuthService(t, &fakeAuthRepository{user: port.AuthUser{
		Auth:  entity.TUserAuth{Id: 7, UserInfoId: 8, Username: "user@example.com", Password: string(hash)},
		Info:  entity.TUserInfo{Id: 8, Email: "user@example.com", Nickname: "user"},
		Roles: []string{"user"},
	}})
	dto, err := service.Authenticate(context.Background(), modelUser("user@example.com", "password"))
	if err != nil || dto == nil {
		t.Fatalf("authentication failed: dto=%+v err=%v", dto, err)
	}
	if dto.Id != 7 || len(dto.Roles) != 1 {
		t.Fatalf("unexpected authenticated user: %+v", dto)
	}
}

func TestUserAuthServiceTreatsMissingUserAsBadCredentials(t *testing.T) {
	service := mustUserAuthService(t, &fakeAuthRepository{err: apperrors.NotFound("auth.find_username")})
	dto, err := service.Authenticate(context.Background(), modelUser("missing@example.com", "password"))
	if err != nil || dto != nil {
		t.Fatalf("missing user should be a credential failure: dto=%+v err=%v", dto, err)
	}
}

type userAreaCacheStub struct {
	fakeServiceCache
	areas       map[string]string
	userArea    string
	userAreaErr error
}

func (f userAreaCacheStub) Get(context.Context, string) (string, error) {
	return f.userArea, f.userAreaErr
}

func (f userAreaCacheStub) HGetAll(context.Context, string) (map[string]string, error) {
	return f.areas, nil
}

func TestUserAuthServiceSkipsMalformedVisitorAreas(t *testing.T) {
	cache := userAreaCacheStub{areas: map[string]string{
		"0|0|北京|北京": "3",
		"malformed": "9",
		"0|0|上海|上海": "not-a-number",
	}}
	service, err := NewUserAuthService(UserAuthServiceDeps{
		Repo:    &fakeAuthRepository{},
		Website: fakeBenetnaschInfoService{},
		Cache:   cache,
		Mailer:  fakeServiceMailer{},
		Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/user/areas?type=2", nil)
	ginContext, _ := gin.CreateTestContext(ctx)
	ginContext.Request = req
	result := service.ListUserAreas(serviceTestRequest{ginContextForServiceTest: ginContext})
	if !result.Flag {
		t.Fatalf("expected successful response, got %+v", result)
	}
	areas, ok := result.Data.([]model.UserAreaDTO)
	if !ok {
		t.Fatalf("unexpected response data type: %T", result.Data)
	}
	if len(areas) != 1 || areas[0].Name != "北京" || areas[0].Value != 3 {
		t.Fatalf("unexpected areas: %+v", areas)
	}
}

func TestUserAuthServiceRejectsMalformedUserAreaCache(t *testing.T) {
	service, err := NewUserAuthService(UserAuthServiceDeps{
		Repo:    &fakeAuthRepository{},
		Website: fakeBenetnaschInfoService{},
		Cache:   userAreaCacheStub{userArea: "not-json"},
		Mailer:  fakeServiceMailer{},
		Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/admin/user/areas?type=1", nil)
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = req
	result := service.ListUserAreas(serviceTestRequest{ginContextForServiceTest: ginContext})
	if result.Flag {
		t.Fatalf("malformed cached user area should fail: %+v", result)
	}
}

func modelUser(username, password string) model.UserVO {
	return model.UserVO{Username: username, Password: password}
}
