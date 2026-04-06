package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
	"fmt"
	"strconv"
)

type TalkRepo interface {
	ListTalks(current, size int) []*model.TalkDTO
	GetTalkById(talkId int) (talk model.TalkDTO)
	ListTalksAdmin(current, size int, vo *model.ConditionVO) []*model.TalkAdminDTO
	GetTalkByIdAdmin(talkId int) (talkAdmin model.TalkAdminDTO)
}

type MyTalkRepo struct {
}

func (t *MyTalkRepo) ListTalks(current, size int) []*model.TalkDTO {
	s := fmt.Sprintf(pgsql.ListTalks, size, (current-1)*size)
	engine := ormInit.GetEngine()
	var talks []*model.TalkDTO
	err := engine.SQL(s).Find(&talks)
	if err != nil {
		zlog.Error(err.Error())
	}

	return talks
}

func (t *MyTalkRepo) GetTalkById(talkId int) (talk model.TalkDTO) {
	s := fmt.Sprintf(pgsql.GetTalkById, talkId)
	engine := ormInit.GetEngine()
	_, err := engine.SQL(s).Get(&talk)
	if err != nil {
		zlog.Error(err.Error())
	}

	return talk
}

func (t *MyTalkRepo) ListTalksAdmin(current, size int, vo *model.ConditionVO) []*model.TalkAdminDTO {
	s := ""
	if vo.Status != 0 {
		s += " where t.status = " + strconv.Itoa(vo.Status)
	}
	s = fmt.Sprintf(pgsql.ListTalksAdmin, s, size, (current-1)*size)
	engine := ormInit.GetEngine()
	var talksAdmin []*model.TalkAdminDTO
	err := engine.SQL(s).Find(&talksAdmin)
	if err != nil {
		zlog.Error(err.Error())
	}

	return talksAdmin
}

func (t *MyTalkRepo) GetTalkByIdAdmin(talkId int) (talkAdmin model.TalkAdminDTO) {
	s := fmt.Sprintf(pgsql.GetTalkByIdAdmin, talkId)
	engine := ormInit.GetEngine()
	_, err := engine.SQL(s).Get(&talkAdmin)
	if err != nil {
		zlog.Error(err.Error())
	}
	return talkAdmin
}
