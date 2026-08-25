package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"context"
	"github.com/gin-gonic/gin"
	"strconv"
	"xorm.io/xorm"
)

type CommentService interface {
	ListTopSixComments() model.ResultVO
	ListComments(c *gin.Context) model.ResultVO
	SaveComment(c *gin.Context) model.ResultVO
	ListRepliesByCommentId(c *gin.Context) model.ResultVO
	ListCommentBackDTO(c *gin.Context) model.ResultVO
	UpdateCommentsReview(c *gin.Context) model.ResultVO
	DeleteComments(c *gin.Context) model.ResultVO
	checkComment(ctx context.Context, vo model.CommentVO) string
}

type MyCommentService struct{}

func (c *MyCommentService) ListTopSixComments() model.ResultVO {
	return model.ResultOkWithData(commentRepo.ListTopSixComments())
}

func (c *MyCommentService) ListComments(ctx *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(ctx.Query("current"))
	if err != nil {
		zlog.Error(err.Error())
	}
	size, err := strconv.Atoi(ctx.Query("size"))
	if err != nil {
		zlog.Error(err.Error())
	}
	var commentVO model.CommentVO
	err = ctx.ShouldBind(&commentVO)
	if err != nil {
		zlog.Error(err.Error())
	}
	sql := "select count(0) from t_comment where type = ? and is_review = ?"
	args := []interface{}{commentVO.Type, shared.TRUE}
	if commentVO.TopicId != "" {
		topicID, parseErr := strconv.Atoi(commentVO.TopicId)
		if parseErr != nil {
			return model.ResultFailWithMessage("参数校验异常")
		}
		sql += " and topic_id = ?"
		args = append(args, topicID)
	}
	var count int
	_, err = ormInit.GetEngine().Context(ctx.Request.Context()).SQL(sql, args...).Get(&count)
	if err != nil {
		zlog.Error(err.Error())
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{})
	}
	commentData := commentRepo.ListComments(current, size, &commentVO)
	if len(commentData) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{})
	}

	var commentIds []int
	for _, v := range commentData {
		commentIds = append(commentIds, v.Id)
	}
	replyData := commentRepo.ListReplies(commentIds)
	replyMap := make(map[int][]model.ReplyDTO)
	for _, v := range replyData {
		replyMap[v.ParentId] = append(replyMap[v.ParentId], *v)
	}
	for _, v := range commentData {
		v.ReplyDTOs = replyMap[v.Id]
	}
	if len(commentData) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: commentData, Count: count})
}

func (c *MyCommentService) SaveComment(ctx *gin.Context) model.ResultVO {
	var commentVO model.CommentVO
	if err := ctx.ShouldBind(&commentVO); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	s := c.checkComment(ctx.Request.Context(), commentVO)
	if s != "" {
		return model.ResultFailWithMessage(s)
	}
	websiteConfigResult := benetnaschService.GetWebsiteConfig(ctx.Request.Context())
	websiteConfig, ok := websiteConfigResult.Data.(model.WebsiteConfigDTO)
	if !ok {
		return websiteConfigResult
	}
	// TODO过滤敏感词汇
	isCommentReview, _ := strconv.Atoi(strconv.Itoa(websiteConfig.IsCommentReview))
	isReview := 0
	if isCommentReview == shared.TRUE {
		isReview = 0
	}
	if isCommentReview == shared.FALSE {
		isReview = 1
	}
	topicId, err := strconv.Atoi(commentVO.TopicId)
	if err != nil {
		return model.ResultFailWithMessage("参数校验异常")
	}

	value, _ := ctx.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	comment := entity.TComment{
		UserId:         dto.UserInfoId,
		ReplyUserId:    commentVO.ReplyUserId,
		TopicId:        topicId,
		CommentContent: commentVO.CommentContent,
		ParentId:       commentVO.ParentId,
		Type:           commentVO.Type,
		IsReview:       isReview,
	}
	if err := ormInit.WithTx(ctx.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Insert(&comment)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (c *MyCommentService) ListRepliesByCommentId(ctx *gin.Context) model.ResultVO {
	commentId, err := strconv.Atoi(ctx.Param("commentId"))
	if err != nil {
		zlog.Error(err.Error())
	}
	var iDs []int
	iDs = append(iDs, commentId)
	data := commentRepo.ListReplies(iDs)
	return model.ResultOkWithData(data)
}

func (c *MyCommentService) ListCommentBackDTO(ctx *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := ctx.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	count := commentRepo.CountComments(&vo)
	data := commentRepo.ListCommentsAdmin(vo.Current, vo.Size, &vo)
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (c *MyCommentService) UpdateCommentsReview(ctx *gin.Context) model.ResultVO {
	var vo model.ReviewVO
	err := ctx.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var comments []entity.TComment
	for _, v := range vo.Ids {
		comment := entity.TComment{
			Id:       v,
			IsReview: vo.IsReview,
		}
		comments = append(comments, comment)
	}
	if err := ormInit.WithTx(ctx.Request.Context(), func(session *xorm.Session) error {
		for _, v := range comments {
			if _, err := session.ID(v.Id).MustCols("is_review").Update(&v); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (c *MyCommentService) DeleteComments(ctx *gin.Context) model.ResultVO {
	var iDs []int
	err := ctx.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
	}
	_, err = ormInit.GetEngine().Context(ctx.Request.Context()).In("id", iDs).Delete(&entity.TComment{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (c *MyCommentService) checkComment(ctx context.Context, vo model.CommentVO) string {
	engine := ormInit.GetEngine().Context(ctx)
	if len(shared.TypeHM[vo.Type]) == 0 {
		return "参数校验异常"
	}

	if vo.Type == shared.ARTICLE || vo.Type == shared.TALK {
		if vo.TopicId == "" {
			return "参数校验异常"
		} else {
			if vo.Type == shared.ARTICLE {
				var article entity.TArticle
				topicID, parseErr := strconv.Atoi(vo.TopicId)
				if parseErr != nil {
					return "参数校验异常"
				}
				get, err := engine.SQL("select id, user_id from t_article where id = ?", topicID).Get(&article)
				if err != nil || !get {
					return "参数校验异常"
				}
			}
			if vo.Type == shared.TALK {
				var talk entity.TTalk
				topicID, parseErr := strconv.Atoi(vo.TopicId)
				if parseErr != nil {
					return "参数校验异常"
				}
				get, err := engine.SQL("select id, user_id from t_talk where id = ?", topicID).Get(&talk)
				if err != nil || !get {
					return "参数校验异常"
				}
			}
		}
	}

	if (vo.Type == shared.LINK || vo.Type == shared.ABOUTS || vo.Type == shared.MESSAGE) && vo.TopicId != "" {
		return "参数校验异常"
	}

	if vo.ParentId == 0 && vo.ReplyUserId != 0 {
		return "参数校验异常"
	}

	if vo.ParentId != 0 {
		var parentComment entity.TComment
		get, err := engine.SQL("select id, parent_id, type from t_comment where id = ?", vo.ParentId).Get(&parentComment)
		if err != nil || !get {
			return "参数校验异常"
		}
		if parentComment.ParentId != 0 {
			return "参数校验异常"
		}
		if vo.Type != parentComment.Type {
			return "参数校验异常"
		}
		if vo.ReplyUserId == 0 {
			return "参数校验异常"
		} else {
			var userInfo entity.TUserInfo
			get, err := engine.SQL("select id from t_user_info where id = ?", vo.ReplyUserId).Get(&userInfo)
			if err != nil || !get {
				return "参数校验异常"
			}
		}
	}
	return ""
}
