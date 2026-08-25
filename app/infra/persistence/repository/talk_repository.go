package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
)

type TalkRepo interface {
	ListTalks(current, size int) []*model.TalkDTO
	GetTalkById(talkId int) (talk model.TalkDTO)
	ListTalksAdmin(current, size int, vo *model.ConditionVO) []*model.TalkAdminDTO
	GetTalkByIdAdmin(talkId int) (talkAdmin model.TalkAdminDTO)
}

type MyTalkRepo struct{}

func (t *MyTalkRepo) ListTalks(current, size int) []*model.TalkDTO {
	limit, offset := pgsql.Page(current, size)
	var talks []*model.TalkDTO
	if err := ormInit.GetEngine().SQL(pgsql.ListTalks, limit, offset).Find(&talks); err != nil {
		zlog.Error("list talks: " + err.Error())
	}
	return talks
}

func (t *MyTalkRepo) GetTalkById(talkID int) model.TalkDTO {
	var talk model.TalkDTO
	if _, err := ormInit.GetEngine().SQL(pgsql.GetTalkById, talkID).Get(&talk); err != nil {
		zlog.Error("get talk: " + err.Error())
	}
	return talk
}

func (t *MyTalkRepo) ListTalksAdmin(current, size int, vo *model.ConditionVO) []*model.TalkAdminDTO {
	limit, offset := pgsql.Page(current, size)
	query := "SELECT t.id, nickname, avatar, content, images, t.is_top, t.status, t.create_time FROM t_talk t JOIN t_user_info ui ON t.user_id = ui.id"
	args := []interface{}{}
	if vo.Status != 0 {
		query += " WHERE t.status = ?"
		args = append(args, vo.Status)
	}
	query += " ORDER BY t.is_top DESC, t.id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var talks []*model.TalkAdminDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&talks); err != nil {
		zlog.Error("list admin talks: " + err.Error())
	}
	return talks
}

func (t *MyTalkRepo) GetTalkByIdAdmin(talkID int) model.TalkAdminDTO {
	var talk model.TalkAdminDTO
	if _, err := ormInit.GetEngine().SQL(pgsql.GetTalkByIdAdmin, talkID).Get(&talk); err != nil {
		zlog.Error("get admin talk: " + err.Error())
	}
	return talk
}
