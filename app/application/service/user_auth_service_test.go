package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"context"
	"testing"

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

func modelUser(username, password string) model.UserVO {
	return model.UserVO{Username: username, Password: password}
}
