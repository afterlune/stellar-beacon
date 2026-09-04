package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
	"strconv"
	"strings"
)

type CommentService interface {
	ListTopSixComments(ctx context.Context) port.ResultVO
	ListComments(c port.Request) port.ResultVO
	SaveComment(c port.Request) port.ResultVO
	ListRepliesByCommentId(c port.Request) port.ResultVO
	ListCommentBackDTO(c port.Request) port.ResultVO
	UpdateCommentsReview(c port.Request) port.ResultVO
	DeleteComments(c port.Request) port.ResultVO
}

type MyCommentService struct {
	repo    port.CommentRepository
	website BenetnaschInfoService
}

func NewCommentService(deps CommentServiceDeps) (*MyCommentService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyCommentService{repo: deps.Repo, website: deps.Website}, nil
}

func (c *MyCommentService) commentRepository() port.CommentRepository {
	return c.repo
}

func (c *MyCommentService) websiteService() BenetnaschInfoService {
	return c.website
}

func (c *MyCommentService) ListTopSixComments(ctx context.Context) port.ResultVO {
	data, err := c.commentRepository().ListTopSixComments(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(data)
}

func (c *MyCommentService) ListComments(ctx port.Request) port.ResultVO {
	current, err := strconv.Atoi(ctx.Query("current"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(ctx.Query("size"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	var commentVO port.CommentVO
	if err := ctx.Bind(&commentVO); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.CommentFilter{Current: current, Size: size, Type: commentVO.Type}
	if commentVO.TopicId != "" {
		topicID, err := strconv.Atoi(commentVO.TopicId)
		if err != nil {
			return port.ResultFailWithMessage("参数校验异常")
		}
		filter.TopicID = &topicID
	}
	commentData, count, err := c.commentRepository().ListComments(ctx.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 || len(commentData) == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	commentIDs := make([]int, 0, len(commentData))
	for _, comment := range commentData {
		commentIDs = append(commentIDs, comment.Id)
	}
	replyData, err := c.commentRepository().ListReplies(ctx.Context(), commentIDs)
	if err != nil {
		return port.ResultFromError(err)
	}
	replyMap := make(map[int][]port.ReplyDTO)
	for _, reply := range replyData {
		replyMap[reply.ParentId] = append(replyMap[reply.ParentId], *reply)
	}
	for _, comment := range commentData {
		comment.ReplyDTOs = replyMap[comment.Id]
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: commentData, Count: count})
}

func (c *MyCommentService) SaveComment(ctx port.Request) port.ResultVO {
	var commentVO port.CommentVO
	if err := ctx.Bind(&commentVO); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.checkComment(ctx.Context(), commentVO); err != nil {
		if apperrors.IsKind(err, apperrors.KindValidation) {
			return port.ResultFailWithMessage("参数校验异常")
		}
		return port.ResultFromError(err)
	}
	websiteConfigResult := c.websiteService().GetWebsiteConfig(ctx.Context())
	websiteConfig, ok := websiteConfigResult.Data.(port.WebsiteConfigDTO)
	if !ok {
		return websiteConfigResult
	}
	isReview := 0
	if websiteConfig.IsCommentReview == support.False {
		isReview = 1
	}
	topicID := 0
	if commentVO.TopicId != "" {
		var err error
		topicID, err = strconv.Atoi(commentVO.TopicId)
		if err != nil {
			return port.ResultFailWithMessage("参数校验异常")
		}
	}
	value, ok := ctx.Get("userInfo")
	if !ok {
		return port.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "comment.user", nil))
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok {
		return port.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "comment.user", nil))
	}
	comment := port.TComment{
		UserId:         dto.UserInfoId,
		ReplyUserId:    commentVO.ReplyUserId,
		TopicId:        topicID,
		CommentContent: commentVO.CommentContent,
		ParentId:       commentVO.ParentId,
		Type:           commentVO.Type,
		IsReview:       isReview,
	}
	if err := c.commentRepository().Create(ctx.Context(), comment); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

// PublishAgentComment is the context-native application use case used after
// a human approves an autonomous candidate. It deliberately creates an
// already-reviewed comment and is not exposed as an HTTP handler.
func (c *MyCommentService) PublishAgentComment(ctx context.Context, userID, articleID int, content string) (int, error) {
	if c == nil || c.repo == nil {
		return 0, apperrors.Unavailable("comment.agent_publish", nil)
	}
	content = strings.TrimSpace(content)
	if userID <= 0 || articleID <= 0 || content == "" || len([]rune(content)) > 100_000 {
		return 0, apperrors.Invalid("comment.agent_publish", "agent comment input is invalid")
	}
	if err := c.repo.ValidateTarget(ctx, support.Article, articleID); err != nil {
		return 0, err
	}
	comment := port.TComment{
		UserId:         userID,
		TopicId:        articleID,
		CommentContent: content,
		Type:           support.Article,
		IsReview:       1,
	}
	if err := c.repo.Create(ctx, comment); err != nil {
		return 0, err
	}
	return comment.Id, nil
}

func (c *MyCommentService) ListRepliesByCommentId(ctx port.Request) port.ResultVO {
	commentID, err := strconv.Atoi(ctx.Param("commentId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	data, err := c.commentRepository().ListReplies(ctx.Context(), []int{commentID})
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(data)
}

func (c *MyCommentService) ListCommentBackDTO(ctx port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := ctx.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.CommentFilter{
		Current:  vo.Current,
		Size:     vo.Size,
		Keywords: vo.Keywords,
		Type:     vo.Type,
		IsReview: vo.IsReview,
	}
	count, err := c.commentRepository().CountComments(ctx.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	data, err := c.commentRepository().ListCommentsAdmin(ctx.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: data, Count: int(count)})
}

func (c *MyCommentService) UpdateCommentsReview(ctx port.Request) port.ResultVO {
	var vo port.ReviewVO
	if err := ctx.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.commentRepository().Review(ctx.Context(), vo.Ids, vo.IsReview); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (c *MyCommentService) DeleteComments(ctx port.Request) port.ResultVO {
	var ids []int
	if err := ctx.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.commentRepository().Delete(ctx.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (c *MyCommentService) checkComment(ctx context.Context, vo port.CommentVO) error {
	if len(support.TypeHM[vo.Type]) == 0 {
		return apperrors.Invalid("comment.validate", "invalid comment type")
	}
	if vo.Type == support.Article || vo.Type == support.Talk {
		if vo.TopicId == "" {
			return apperrors.Invalid("comment.validate", "topic is required")
		}
		topicID, err := strconv.Atoi(vo.TopicId)
		if err != nil {
			return apperrors.Invalid("comment.validate", "invalid topic")
		}
		if err := c.commentRepository().ValidateTarget(ctx, vo.Type, topicID); err != nil {
			return err
		}
	}
	if (vo.Type == support.Link || vo.Type == support.Abouts || vo.Type == support.Message) && vo.TopicId != "" {
		return apperrors.Invalid("comment.validate", "topic must be empty")
	}
	if vo.ParentId == 0 && vo.ReplyUserId != 0 {
		return apperrors.Invalid("comment.validate", "reply user requires parent")
	}
	if vo.ParentId != 0 {
		if vo.ReplyUserId == 0 {
			return apperrors.Invalid("comment.validate", "reply user is required")
		}
		return c.commentRepository().ValidateReply(ctx, vo.Type, vo.ParentId, vo.ReplyUserId)
	}
	return nil
}

var _ CommentService = (*MyCommentService)(nil)
