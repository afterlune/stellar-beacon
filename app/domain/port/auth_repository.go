package port

import "context"

type AuthUser struct {
	Auth  TUserAuth
	Info  TUserInfo
	Roles []string
}

type AuthRepository interface {
	ListUsers(ctx context.Context, filter UserFilter) ([]*UserAdmin, int64, error)
	FindByUsername(ctx context.Context, username string) (AuthUser, error)
	FindByID(ctx context.Context, id int) (TUserAuth, error)
	CreateUser(ctx context.Context, info TUserInfo, auth TUserAuth, roleID int) error
	UpdatePassword(ctx context.Context, username, password string) error
	UpdatePasswordByID(ctx context.Context, id int, password string) error
	UpdateLoginMetadata(ctx context.Context, user TUserAuth) error
}
