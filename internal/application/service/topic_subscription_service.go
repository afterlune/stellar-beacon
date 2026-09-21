package service

import (
	"strings"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type TopicSubscriptionService interface {
	Subscribe(c *gin.Context) model.ResultVO
	Unsubscribe(c *gin.Context) model.ResultVO
	SetMuted(c *gin.Context) model.ResultVO
	ListSubscriptions(c *gin.Context) model.ResultVO
	ListFeed(c *gin.Context) model.ResultVO
}

type MyTopicSubscriptionService struct {
	repo port.TopicSubscriptionRepository
}

func NewTopicSubscriptionService(repo port.TopicSubscriptionRepository) *MyTopicSubscriptionService {
	return &MyTopicSubscriptionService{repo: repo}
}

// topicMuteRequest keeps the mute flag explicit: an absent field must not be
// read as "unmute".
type topicMuteRequest struct {
	Muted *int `json:"muted" form:"muted"`
}

func topicParams(c *gin.Context) (string, string, bool) {
	topicType := strings.ToLower(strings.TrimSpace(c.Param("topicType")))
	topicKey := strings.TrimSpace(c.Param("topicKey"))
	if !port.ValidTopicType(topicType) || topicKey == "" {
		return "", "", false
	}
	return topicType, topicKey, true
}

func (s *MyTopicSubscriptionService) Subscribe(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	topicType, topicKey, ok := topicParams(c)
	if !ok {
		return model.ResultFailWithMessage("话题参数不正确")
	}
	if err := s.repo.Subscribe(c.Request.Context(), user.UserInfoId, topicType, topicKey); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultFailWithMessage("话题不存在或还没有公开内容")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyTopicSubscriptionService) Unsubscribe(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	topicType, topicKey, ok := topicParams(c)
	if !ok {
		return model.ResultFailWithMessage("话题参数不正确")
	}
	if err := s.repo.Unsubscribe(c.Request.Context(), user.UserInfoId, topicType, topicKey); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyTopicSubscriptionService) SetMuted(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	topicType, topicKey, ok := topicParams(c)
	if !ok {
		return model.ResultFailWithMessage("话题参数不正确")
	}
	var vo topicMuteRequest
	if err := c.ShouldBind(&vo); err != nil || vo.Muted == nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.repo.SetMuted(c.Request.Context(), user.UserInfoId, topicType, topicKey, *vo.Muted == 1); err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultFailWithMessage("还没有订阅这个话题")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyTopicSubscriptionService) ListSubscriptions(c *gin.Context) model.ResultVO {
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

func (s *MyTopicSubscriptionService) ListFeed(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	records, count, err := s.repo.ListTopicFeed(c.Request.Context(), user.UserInfoId, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size})
}
