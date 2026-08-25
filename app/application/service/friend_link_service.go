package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/shared"
	"container/list"
	"context"

	"github.com/gin-gonic/gin"
)

type FriendLinkService interface {
	ListFriendLinks() model.ResultVO
	ListFriendLinkDTO(c *gin.Context) model.ResultVO
	SaveOrUpdateFriendLink(c *gin.Context) model.ResultVO
	DeleteFriendLink(c *gin.Context) model.ResultVO
}

type MyFriendLinkService struct{ repo port.FriendLinkRepository }

func NewFriendLinkService(repo port.FriendLinkRepository) *MyFriendLinkService {
	return &MyFriendLinkService{repo: repo}
}

func (f *MyFriendLinkService) friendLinkRepository() port.FriendLinkRepository {
	if f.repo != nil {
		return f.repo
	}
	return friendLinkRepo
}

func (f *MyFriendLinkService) ListFriendLinks() model.ResultVO {
	links, err := f.friendLinkRepository().ListPublic(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.FriendLinkDTO
	shared.StructCopy(links, &dtos)
	return model.ResultOkWithData(dtos)
}

func (f *MyFriendLinkService) ListFriendLinkDTO(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	links, count, err := f.friendLinkRepository().ListAdmin(c.Request.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.FriendLinkAdminDTO
	shared.StructCopy(links, &dtos)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)})
}

func (f *MyFriendLinkService) SaveOrUpdateFriendLink(c *gin.Context) model.ResultVO {
	var vo model.FriendLinkVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	link := entity.TFriendLink{Id: vo.Id, LinkName: vo.LinkName, LinkAvatar: vo.LinkAvatar, LinkAddress: vo.LinkAddress, LinkIntro: vo.LinkIntro}
	if err := f.friendLinkRepository().SaveOrUpdate(c.Request.Context(), link); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (f *MyFriendLinkService) DeleteFriendLink(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := f.friendLinkRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}
