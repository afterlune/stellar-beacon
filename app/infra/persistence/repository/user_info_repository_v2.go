package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/row"
	"context"

	"xorm.io/xorm"
)

var _ port.UserInfoRepository = (*MyUserInfoRepo)(nil)

type MyUserInfoRepo struct{ engine *xorm.Engine }

func NewUserInfoRepo(engine *xorm.Engine) *MyUserInfoRepo { return &MyUserInfoRepo{engine: engine} }

func (r *MyUserInfoRepo) UpdateProfile(ctx context.Context, id int, nickname, intro, website string) error {
	return repoTx(r.engine, ctx, "user_info.profile", func(session *xorm.Session) error {
		_, err := session.ID(id).Cols("nickname", "intro", "website").Update(&row.TUserInfo{
			Id: id, Nickname: nickname, Intro: intro, Website: website,
		})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateAvatar(ctx context.Context, id int, avatar string) error {
	return repoTx(r.engine, ctx, "user_info.avatar", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("avatar").Update(&row.TUserInfo{Id: id, Avatar: avatar})
		return err
	})
}

func (r *MyUserInfoRepo) GetByID(ctx context.Context, id int) (port.TUserInfo, error) {
	session, err := repoSession(r.engine, ctx, "user_info.get")
	if err != nil {
		return port.TUserInfo{}, err
	}
	var info row.TUserInfo
	found, err := session.ID(id).Get(&info)
	if err != nil {
		return port.TUserInfo{}, apperrors.Unavailable("user_info.get", err)
	}
	if !found {
		return port.TUserInfo{}, apperrors.NotFound("user_info.get")
	}
	return row.FromUserInfo(info), nil
}

func (r *MyUserInfoRepo) UpdateEmail(ctx context.Context, id int, email string) error {
	return repoTx(r.engine, ctx, "user_info.email", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("email").Update(&row.TUserInfo{Id: id, Email: email})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateSubscribe(ctx context.Context, id, subscribe int) error {
	return repoTx(r.engine, ctx, "user_info.subscribe", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("is_subscribe").Update(&row.TUserInfo{Id: id, IsSubscribe: subscribe})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateRole(ctx context.Context, userInfoID int, nickname string, roleIDs []int) error {
	return repoTx(r.engine, ctx, "user_info.role", func(session *xorm.Session) error {
		if _, err := session.ID(userInfoID).MustCols("nickname").Update(&row.TUserInfo{Id: userInfoID, Nickname: nickname}); err != nil {
			return err
		}
		if _, err := session.Where("user_id = ?", userInfoID).Delete(&row.TUserRole{}); err != nil {
			return err
		}
		roles := make([]row.TUserRole, 0, len(roleIDs))
		for _, roleID := range roleIDs {
			roles = append(roles, row.TUserRole{UserId: userInfoID, RoleId: roleID})
		}
		if len(roles) == 0 {
			return nil
		}
		_, err := session.Insert(&roles)
		return err
	})
}

func (r *MyUserInfoRepo) UpdateDisable(ctx context.Context, id, disabled int) error {
	return repoTx(r.engine, ctx, "user_info.disable", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("is_disable").Update(&row.TUserInfo{Id: id, IsDisable: disabled})
		return err
	})
}

func (r *MyUserInfoRepo) FindAuthByUserInfoID(ctx context.Context, id int) (port.TUserAuth, error) {
	session, err := repoSession(r.engine, ctx, "user_info.auth")
	if err != nil {
		return port.TUserAuth{}, err
	}
	var auth row.TUserAuth
	found, err := session.Where("user_info_id = ?", id).Get(&auth)
	if err != nil {
		return port.TUserAuth{}, apperrors.Unavailable("user_info.auth", err)
	}
	if !found {
		return port.TUserAuth{}, apperrors.NotFound("user_info.auth")
	}
	return row.FromUserAuth(auth), nil
}
