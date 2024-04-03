package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

type FriendLinkService interface {
	ListFriendLinks() model.ResultVO
	ListFriendLinkDTO(c *gin.Context) model.ResultVO
	SaveOrUpdateFriendLink(c *gin.Context) model.ResultVO
	DeleteFriendLink(c *gin.Context) model.ResultVO
}

type MyFriendLinkService struct{}

func (f *MyFriendLinkService) ListFriendLinks() model.ResultVO {
	var frilinks []entity.TFriendLink
	err := ormInit.GetEngine().Find(&frilinks)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	var frilinkDTOs []model.FriendLinkDTO
	frilinksjson, err := json.Marshal(frilinks)
	if err != nil {
		zlog.Error(err.Error())
	}

	err = json.Unmarshal(frilinksjson, &frilinkDTOs)
	if err != nil {
		zlog.Error(err.Error())
	}

	return model.ResultOkWithData(frilinkDTOs)
}

func (f *MyFriendLinkService) ListFriendLinkDTO(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var friendLinks []entity.TFriendLink
	if vo.Keywords != "" {
		err = engine.Prepare().Where("link_name like '%"+vo.Keywords+"%'").Limit(vo.Size, vo.Size*(vo.Current-1)).Find(&friendLinks)
	} else {
		err = engine.Prepare().Limit(vo.Size, vo.Size*(vo.Current-1)).Find(&friendLinks)
	}
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	count, err := engine.Count(&entity.TFriendLink{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var dtos []model.FriendLinkAdminDTO
	shared.StructCopy(friendLinks, &dtos)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)})
}

func (f *MyFriendLinkService) SaveOrUpdateFriendLink(c *gin.Context) model.ResultVO {
	var vo model.FriendLinkVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var friendLink entity.TFriendLink
	shared.StructCopy(vo, &friendLink)
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	if friendLink.Id != 0 {
		_, err = session.Prepare().ID(friendLink.Id).Update(&friendLink)
	} else {
		_, err = session.Prepare().Insert(&friendLink)
	}
	if err != nil {
		zlog.Error(err.Error())
		zlog.Unwrap(session.Rollback())
		return model.ResultFail()
	}
	session.Commit()
	return model.ResultOk()
}

func (f *MyFriendLinkService) DeleteFriendLink(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	_, err = ormInit.GetEngine().Prepare().In("id", iDs).Delete(&entity.TFriendLink{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}
