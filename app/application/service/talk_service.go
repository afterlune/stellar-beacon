package service

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
)

type TalkService interface {
	ListTalks(c port.Request) port.ResultVO
	GetTalkById(c port.Request) port.ResultVO
	SaveTalkImages(c port.Request) port.ResultVO
	SaveOrUpdateTalk(c port.Request) port.ResultVO
	DeleteTalks(c port.Request) port.ResultVO
	ListBackTalks(c port.Request) port.ResultVO
	GetBackTalkById(c port.Request) port.ResultVO
}

type MyTalkService struct {
	repo     port.TalkRepository
	comments port.CommentRepository
	storage  port.ObjectStorage
}

func NewTalkService(deps TalkServiceDeps) (*MyTalkService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyTalkService{repo: deps.Repo, comments: deps.Comments, storage: deps.Storage}, nil
}

func (t *MyTalkService) talkRepository() port.TalkRepository {
	return t.repo
}

func (t *MyTalkService) commentRepository() port.CommentRepository {
	return t.comments
}

func (t *MyTalkService) ListTalks(c port.Request) port.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	ctx := c.Context()
	count, err := t.talkRepository().Count(ctx, port.TalkFilter{Status: 1})
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	talks, err := t.talkRepository().List(ctx, current, size)
	if err != nil {
		return port.ResultFromError(err)
	}
	talkIDs := make([]int, 0, len(talks))
	for _, talk := range talks {
		talkIDs = append(talkIDs, talk.Id)
	}
	comments, err := t.commentRepository().ListCommentCountsByTypeAndTopicIDs(ctx, 5, talkIDs)
	if err != nil {
		return port.ResultFromError(err)
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
				slog.Error("decode talk images failed", "error_code", apperrors.SafeCode(err))
				return port.ResultFailWithMessage("说说图片格式不正确")
			}
			talk.Imgs = images
		}
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: talks, Count: count})
}

func (t *MyTalkService) GetTalkById(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("talkId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	talk, err := t.talkRepository().Get(c.Context(), id)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ResultFailWithMessage("说说不存在")
		}
		return port.ResultFromError(err)
	}
	if talk.Images != "" {
		var images []string
		if err := json.Unmarshal([]byte(talk.Images), &images); err != nil {
			return port.ResultFailWithMessage("说说图片格式不正确")
		}
		talk.Imgs = images
	}
	commentCount, err := t.commentRepository().ListCommentCountByTypeAndTopicID(c.Context(), 5, id)
	if err != nil {
		return port.ResultFromError(err)
	}
	talk.CommentCount = commentCount.CommentCount
	return port.ResultOkWithData(talk)
}

func (t *MyTalkService) SaveTalkImages(c port.Request) port.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	ref, err := uploadMultipart(c.Context(), t.storage, file, "talks/")
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(ref.URL)
}

func (t *MyTalkService) SaveOrUpdateTalk(c port.Request) port.ResultVO {
	var vo port.TalkVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	value, ok := c.Get("userInfo")
	if !ok {
		return port.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "talk.user", nil))
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok {
		return port.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "talk.user", nil))
	}
	talk := port.TTalk{Id: vo.Id, Content: vo.Content, Images: vo.Images, IsTop: vo.IsTop, Status: vo.Status, UserId: dto.UserInfoId}
	if err := t.talkRepository().SaveOrUpdate(c.Context(), talk); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

// PublishAgentTalk is the context-native application use case used after a
// human approves an autonomous candidate. It creates a public, non-pinned
// talk without exposing an HTTP or model dependency.
func (t *MyTalkService) PublishAgentTalk(ctx context.Context, userID int, content string) (int, error) {
	if t == nil || t.repo == nil {
		return 0, apperrors.Unavailable("talk.agent_publish", nil)
	}
	content = strings.TrimSpace(content)
	if userID <= 0 || content == "" || len([]rune(content)) > 100_000 {
		return 0, apperrors.Invalid("talk.agent_publish", "agent talk input is invalid")
	}
	talk := port.TTalk{UserId: userID, Content: content, IsTop: 0, Status: 1}
	if err := t.repo.SaveOrUpdate(ctx, talk); err != nil {
		return 0, err
	}
	return talk.Id, nil
}

func (t *MyTalkService) DeleteTalks(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := t.talkRepository().Delete(c.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (t *MyTalkService) ListBackTalks(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.TalkFilter{Status: vo.Status}
	count, err := t.talkRepository().Count(c.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	talks, err := t.talkRepository().ListAdmin(c.Context(), vo.Current, vo.Size, filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	for _, talk := range talks {
		if talk.Images != "" {
			var images []string
			if err := json.Unmarshal([]byte(talk.Images), &images); err != nil {
				return port.ResultFailWithMessage("说说图片格式不正确")
			}
			talk.Imgs = images
		}
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: talks, Count: count})
}

func (t *MyTalkService) GetBackTalkById(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("talkId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	talk, err := t.talkRepository().GetAdmin(c.Context(), id)
	if err != nil {
		return port.ResultFromError(err)
	}
	if talk.Images != "" {
		var images []string
		if err := json.Unmarshal([]byte(talk.Images), &images); err != nil {
			return port.ResultFailWithMessage("说说图片格式不正确")
		}
		talk.Imgs = images
	}
	return port.ResultOkWithData(talk)
}

var _ TalkService = (*MyTalkService)(nil)
