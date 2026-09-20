package service

import (
	"strings"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type FollowService interface {
	Follow(c *gin.Context) model.ResultVO
	Unfollow(c *gin.Context) model.ResultVO
	ListFollowing(c *gin.Context) model.ResultVO
	ListFollowers(c *gin.Context) model.ResultVO
	ListFeed(c *gin.Context) model.ResultVO
	ListNotifications(c *gin.Context) model.ResultVO
	UnreadNotificationCount(c *gin.Context) model.ResultVO
	MarkNotificationsRead(c *gin.Context) model.ResultVO
}

type MyFollowService struct{ repo port.FollowRepository }

func NewFollowService(repo port.FollowRepository) *MyFollowService {
	return &MyFollowService{repo: repo}
}

func (s *MyFollowService) Follow(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	authorID, err := pathID(c, "authorId")
	if err != nil || authorID == user.UserInfoId {
		return model.ResultFailWithMessage("关注对象不正确")
	}
	if err := s.repo.Follow(c.Request.Context(), user.UserInfoId, authorID); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyFollowService) Unfollow(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	authorID, err := pathID(c, "authorId")
	if err != nil || authorID == user.UserInfoId {
		return model.ResultFailWithMessage("关注对象不正确")
	}
	if err := s.repo.Unfollow(c.Request.Context(), user.UserInfoId, authorID); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyFollowService) ListFollowing(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	records, count, err := s.repo.ListFollowing(c.Request.Context(), user.UserInfoId, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size})
}

func (s *MyFollowService) ListFollowers(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	records, count, err := s.repo.ListFollowers(c.Request.Context(), user.UserInfoId, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size})
}

func (s *MyFollowService) ListFeed(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	contentType := strings.ToLower(strings.TrimSpace(c.Query("type")))
	if contentType == "all" {
		contentType = ""
	}
	if contentType != "" && contentType != port.FollowContentArticle && contentType != port.FollowContentTalk {
		return model.ResultFailWithMessage("内容类型不正确")
	}
	records, count, err := s.repo.ListFollowFeed(c.Request.Context(), user.UserInfoId, contentType, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: count, Page: current, PageSize: size})
}

func (s *MyFollowService) ListNotifications(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	group := strings.ToLower(strings.TrimSpace(c.Query("group")))
	if group == "" {
		group = port.NotificationGroupAll
	}
	if !port.ValidNotificationGroup(group) {
		return model.ResultFailWithMessage("通知类型不正确")
	}
	result, err := s.repo.ListNotifications(c.Request.Context(), user.UserInfoId, group, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(struct {
		Records          []port.NotificationItem `json:"records"`
		Count            int                     `json:"count"`
		Page             int                     `json:"page"`
		PageSize         int                     `json:"pageSize"`
		UnreadCount      int                     `json:"unreadCount"`
		TotalUnreadCount int                     `json:"totalUnreadCount"`
		ReadCursor       port.NotificationCursor `json:"readCursor"`
	}{Records: result.Records, Count: result.Count, Page: current, PageSize: size, UnreadCount: result.UnreadCount, TotalUnreadCount: result.TotalUnreadCount, ReadCursor: result.ReadCursor})
}

func (s *MyFollowService) UnreadNotificationCount(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	count, err := s.repo.UnreadNotificationCount(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(map[string]int{"count": count})
}

func (s *MyFollowService) MarkNotificationsRead(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	cursor := port.NotificationCursor{}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&cursor); err != nil {
			return model.ResultFailWithMessage("参数格式不正确")
		}
	}
	if err := s.repo.MarkNotificationsRead(c.Request.Context(), user.UserInfoId, cursor); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(map[string]int{"unreadCount": 0})
}
