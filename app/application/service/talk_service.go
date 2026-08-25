package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/shared"
	"container/list"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

type TalkService interface {
	ListTalks(c *gin.Context) model.ResultVO
	GetTalkById(c *gin.Context) model.ResultVO
	SaveTalkImages(c *gin.Context) model.ResultVO
	SaveOrUpdateTalk(c *gin.Context) model.ResultVO
	DeleteTalks(c *gin.Context) model.ResultVO
	ListBackTalks(c *gin.Context) model.ResultVO
	GetBackTalkById(c *gin.Context) model.ResultVO
}

type MyTalkService struct {
	repo     port.TalkRepository
	comments port.CommentRepository
}

func NewTalkService(repo port.TalkRepository, comments ...port.CommentRepository) *MyTalkService {
	service := &MyTalkService{repo: repo}
	if len(comments) > 0 {
		service.comments = comments[0]
	}
	return service
}

func (t *MyTalkService) talkRepository() port.TalkRepository {
	if t.repo != nil {
		return t.repo
	}
	return talkRepo
}

func (t *MyTalkService) commentRepository() port.CommentRepository {
	if t.comments != nil {
		return t.comments
	}
	return commentRepo
}

func (t *MyTalkService) ListTalks(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	ctx := c.Request.Context()
	count, err := t.talkRepository().Count(ctx, port.TalkFilter{Status: 1})
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	talks, err := t.talkRepository().List(ctx, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	talkIDs := make([]int, 0, len(talks))
	for _, talk := range talks {
		talkIDs = append(talkIDs, talk.Id)
	}
	comments, err := t.commentRepository().ListCommentCountsByTypeAndTopicIDs(ctx, 5, talkIDs)
	if err != nil {
		return model.ResultFromError(err)
	}
	commentCounts := make(map[int]int, len(comments))
	for _, comment := range comments {
		commentCounts[comment.Id] = comment.CommentCount
	}
	for _, talk := range talks {
		talk.CommentCount = commentCounts[talk.Id]
		if talk.Images != "" {
			var images []string
			if err := json.Unmarshal([]byte(talk.Images), &images); err != nil {
				slog.Error("decode talk images failed", "error", err)
				return model.ResultFailWithMessage("说说图片格式不正确")
			}
			talk.Imgs = images
		}
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: count})
}

func (t *MyTalkService) GetTalkById(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("talkId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	talk, err := t.talkRepository().Get(c.Request.Context(), id)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultFailWithMessage("说说不存在")
		}
		return model.ResultFromError(err)
	}
	if talk.Images != "" {
		var images []string
		if err := json.Unmarshal([]byte(talk.Images), &images); err != nil {
			return model.ResultFailWithMessage("说说图片格式不正确")
		}
		talk.Imgs = images
	}
	commentCount, err := t.commentRepository().ListCommentCountByTypeAndTopicID(c.Request.Context(), 5, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	talk.CommentCount = commentCount.CommentCount
	return model.ResultOkWithData(talk)
}

func (t *MyTalkService) SaveTalkImages(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	fileURL := oss.Upload(file, "talks/")
	return model.ResultOkWithData(shared.FILEURL + fileURL)
}

func (t *MyTalkService) SaveOrUpdateTalk(c *gin.Context) model.ResultVO {
	var vo model.TalkVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "talk.user", nil))
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "talk.user", nil))
	}
	talk := entity.TTalk{Id: vo.Id, Content: vo.Content, Images: vo.Images, IsTop: vo.IsTop, Status: vo.Status, UserId: dto.UserInfoId}
	if err := t.talkRepository().SaveOrUpdate(c.Request.Context(), talk); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (t *MyTalkService) DeleteTalks(c *gin.Context) model.ResultVO {
	var values []string
	if err := c.ShouldBind(&values); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	ids := make([]int, 0, len(values))
	for _, value := range values {
		id, err := strconv.Atoi(value)
		if err != nil {
			return model.ResultFailWithMessage("参数格式不正确")
		}
		ids = append(ids, id)
	}
	if err := t.talkRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (t *MyTalkService) ListBackTalks(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.TalkFilter{Status: vo.Status}
	count, err := t.talkRepository().Count(c.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	talks, err := t.talkRepository().ListAdmin(c.Request.Context(), vo.Current, vo.Size, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	for _, talk := range talks {
		if talk.Images != "" {
			var images []string
			if err := json.Unmarshal([]byte(talk.Images), &images); err != nil {
				return model.ResultFailWithMessage("说说图片格式不正确")
			}
			talk.Imgs = images
		}
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: count})
}

func (t *MyTalkService) GetBackTalkById(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("talkId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	talk, err := t.talkRepository().GetAdmin(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	if talk.Images != "" {
		var images []string
		if err := json.Unmarshal([]byte(talk.Images), &images); err != nil {
			return model.ResultFailWithMessage("说说图片格式不正确")
		}
		talk.Imgs = images
	}
	return model.ResultOkWithData(talk)
}

var _ TalkService = (*MyTalkService)(nil)
