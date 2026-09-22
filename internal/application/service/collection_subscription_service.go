package service

import (
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type CollectionSubscriptionService interface {
	Subscribe(c *gin.Context) model.ResultVO
	Unsubscribe(c *gin.Context) model.ResultVO
	SetMuted(c *gin.Context) model.ResultVO
	GetStatus(c *gin.Context) model.ResultVO
	ListSubscriptions(c *gin.Context) model.ResultVO
	ListFeed(c *gin.Context) model.ResultVO
}

type MyCollectionSubscriptionService struct {
	repo port.CollectionSubscriptionRepository
}

func NewCollectionSubscriptionService(repo port.CollectionSubscriptionRepository) *MyCollectionSubscriptionService {
	return &MyCollectionSubscriptionService{repo: repo}
}

type collectionMuteRequest struct {
	Muted *int `json:"muted" form:"muted"`
}

func (s *MyCollectionSubscriptionService) Subscribe(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.repo.Subscribe(c.Request.Context(), user.UserInfoId, collectionID); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultFailWithMessage("书单不存在或不可订阅")
		}
		if apperrors.IsKind(err, apperrors.KindValidation) {
			return model.ResultFailWithMessage("不能订阅自己的书单")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyCollectionSubscriptionService) Unsubscribe(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.repo.Unsubscribe(c.Request.Context(), user.UserInfoId, collectionID); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyCollectionSubscriptionService) SetMuted(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo collectionMuteRequest
	if err := c.ShouldBind(&vo); err != nil || vo.Muted == nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.repo.SetMuted(c.Request.Context(), user.UserInfoId, collectionID, *vo.Muted == 1); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultFailWithMessage("还没有订阅这个书单")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyCollectionSubscriptionService) GetStatus(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	status, err := s.repo.GetStatus(c.Request.Context(), user.UserInfoId, collectionID)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(status)
}

func (s *MyCollectionSubscriptionService) ListSubscriptions(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	records, count, err := s.repo.ListSubscriptions(c.Request.Context(), user.UserInfoId, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size})
}

func (s *MyCollectionSubscriptionService) ListFeed(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	records, count, err := s.repo.ListFeed(c.Request.Context(), user.UserInfoId, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size})
}
