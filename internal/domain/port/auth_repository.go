package port

import (
	"context"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
)

type AuthUser struct {
	Auth  entity.TUserAuth
	Info  entity.TUserInfo
	Roles []string
}

type AuthRepository interface {
	ListUsers(ctx context.Context, filter UserFilter) ([]*UserAdmin, int64, error)
	FindByUsername(ctx context.Context, username string) (AuthUser, error)
	FindByID(ctx context.Context, id int) (entity.TUserAuth, error)
	CreateUser(ctx context.Context, info entity.TUserInfo, auth entity.TUserAuth, roleID int) error
	UpdatePassword(ctx context.Context, username, password string) error
	UpdatePasswordByID(ctx context.Context, id int, password string) error
	UpdateLoginMetadata(ctx context.Context, user entity.TUserAuth) error
	ListAreaSources(ctx context.Context) ([]UserAreaSource, error)
}

type UserAreaSource struct {
	IpSource string `json:"ipSource" xorm:"ip_source"`
}
