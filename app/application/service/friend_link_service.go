package service

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
)

type FriendLinkService interface {
	ListFriendLinks(ctx context.Context) port.ResultVO
	ListFriendLinkDTO(c port.Request) port.ResultVO
	SaveOrUpdateFriendLink(c port.Request) port.ResultVO
	DeleteFriendLink(c port.Request) port.ResultVO
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

func (f *MyFriendLinkService) ListFriendLinks(ctx context.Context) port.ResultVO {
	links, err := f.friendLinkRepository().ListPublic(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dtos []port.FriendLinkDTO
	support.StructCopy(links, &dtos)
	return port.ResultOkWithData(dtos)
}

func (f *MyFriendLinkService) ListFriendLinkDTO(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	links, count, err := f.friendLinkRepository().ListAdmin(c.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dtos []port.FriendLinkAdminDTO
	support.StructCopy(links, &dtos)
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: dtos, Count: int(count)})
}

func (f *MyFriendLinkService) SaveOrUpdateFriendLink(c port.Request) port.ResultVO {
	var vo port.FriendLinkVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	link := port.TFriendLink{Id: vo.Id, LinkName: vo.LinkName, LinkAvatar: vo.LinkAvatar, LinkAddress: vo.LinkAddress, LinkIntro: vo.LinkIntro}
	if err := f.friendLinkRepository().SaveOrUpdate(c.Context(), link); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (f *MyFriendLinkService) DeleteFriendLink(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := f.friendLinkRepository().Delete(c.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}
