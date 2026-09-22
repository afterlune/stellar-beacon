package repository

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"

	"xorm.io/xorm"
)

var _ port.UserInfoRepository = (*MyUserInfoRepo)(nil)

type MyUserInfoRepo struct{ engine *xorm.Engine }

func NewUserInfoRepo(engine *xorm.Engine) *MyUserInfoRepo { return &MyUserInfoRepo{engine: engine} }

func (r *MyUserInfoRepo) UpdateProfile(ctx context.Context, id int, nickname, intro, website string) error {
	return repoTx(r.engine, ctx, "user_info.profile", func(session *xorm.Session) error {
		_, err := session.ID(id).Cols("nickname", "intro", "website").Update(&entity.TUserInfo{
			Id: id, Nickname: nickname, Intro: intro, Website: website,
		})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateAvatar(ctx context.Context, id int, avatar string) error {
	return repoTx(r.engine, ctx, "user_info.avatar", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("avatar").Update(&entity.TUserInfo{Id: id, Avatar: avatar})
		return err
	})
}

func (r *MyUserInfoRepo) GetByID(ctx context.Context, id int) (entity.TUserInfo, error) {
	session, err := repoSession(r.engine, ctx, "user_info.get")
	if err != nil {
		return entity.TUserInfo{}, err
	}
	var info entity.TUserInfo
	found, err := session.ID(id).Get(&info)
	if err != nil {
		return entity.TUserInfo{}, apperrors.Unavailable("user_info.get", err)
	}
	if !found {
		return entity.TUserInfo{}, apperrors.NotFound("user_info.get")
	}
	return info, nil
}

func (r *MyUserInfoRepo) UpdateEmail(ctx context.Context, id int, email string) error {
	return repoTx(r.engine, ctx, "user_info.email", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("email").Update(&entity.TUserInfo{Id: id, Email: email})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateSubscribe(ctx context.Context, id, subscribe int) error {
	return repoTx(r.engine, ctx, "user_info.subscribe", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("is_subscribe").Update(&entity.TUserInfo{Id: id, IsSubscribe: subscribe})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateNotifyComment(ctx context.Context, id, notify int) error {
	return repoTx(r.engine, ctx, "user_info.notify_comment", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("notify_comment").Update(&entity.TUserInfo{Id: id, NotifyComment: notify})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateNotifyInteraction(ctx context.Context, id, notify int) error {
	return repoTx(r.engine, ctx, "user_info.notify_interaction", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("notify_interaction").Update(&entity.TUserInfo{Id: id, NotifyInteraction: notify})
		return err
	})
}
func (r *MyUserInfoRepo) UpdateNotifyTopic(ctx context.Context, id, notify int) error {
	return repoTx(r.engine, ctx, "user_info.notify_topic", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("notify_topic").Update(&entity.TUserInfo{Id: id, NotifyTopic: notify})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateNotifyCollection(ctx context.Context, id, notify int) error {
	return repoTx(r.engine, ctx, "user_info.notify_collection", func(session *xorm.Session) error {
		_, err := session.ID(id).MustCols("notify_collection").Update(&entity.TUserInfo{Id: id, NotifyCollection: notify})
		return err
	})
}

func (r *MyUserInfoRepo) UpdateRole(ctx context.Context, userInfoID int, nickname string, roleIDs []int) error {
	return repoTx(r.engine, ctx, "user_info.role", func(session *xorm.Session) error {
		if _, err := session.ID(userInfoID).MustCols("nickname").Update(&entity.TUserInfo{Id: userInfoID, Nickname: nickname}); err != nil {
			return err
		}
		if _, err := session.Where("user_id = ?", userInfoID).Delete(&entity.TUserRole{}); err != nil {
			return err
		}
		roles := make([]entity.TUserRole, 0, len(roleIDs))
		for _, roleID := range roleIDs {
			roles = append(roles, entity.TUserRole{UserId: userInfoID, RoleId: roleID})
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
		_, err := session.ID(id).MustCols("is_disable").Update(&entity.TUserInfo{Id: id, IsDisable: disabled})
		return err
	})
}

func (r *MyUserInfoRepo) FindAuthByUserInfoID(ctx context.Context, id int) (entity.TUserAuth, error) {
	session, err := repoSession(r.engine, ctx, "user_info.auth")
	if err != nil {
		return entity.TUserAuth{}, err
	}
	var auth entity.TUserAuth
	found, err := session.Where("user_info_id = ?", id).Get(&auth)
	if err != nil {
		return entity.TUserAuth{}, apperrors.Unavailable("user_info.auth", err)
	}
	if !found {
		return entity.TUserAuth{}, apperrors.NotFound("user_info.auth")
	}
	return auth, nil
}
