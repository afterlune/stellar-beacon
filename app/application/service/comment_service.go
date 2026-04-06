package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"fmt"
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
	checkComment(vo model.CommentVO) string
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
	sql := ""
	if commentVO.TopicId == "" {
		sql = fmt.Sprintf("select count(0) from t_comment where type = %d and is_review = %d", commentVO.Type, shared.TRUE)
	} else {
		sql = fmt.Sprintf("select count(0) from t_comment where topic_id = %s and type = %d and is_review = %d",
			commentVO.TopicId, commentVO.Type, shared.TRUE)
	}
	var count int
	_, err = ormInit.GetEngine().SQL(sql).Get(&count)
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
	zlog.Unwrap(ctx.ShouldBind(&commentVO))

	s := c.checkComment(commentVO)
	if s != "" {
		return model.ResultFailWithMessage(s)
	}
	websiteConfig := benetnaschService.GetWebsiteConfig().Data.(model.WebsiteConfigDTO)
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
	session := ormInit.GetEngine().NewSession()
	zlog.Unwrap(session.Begin())
	defer session.Close()
	defer func(session *xorm.Session) {
		zlog.Unwrap(session.Close())
	}(session)

	_, err = session.Prepare().Insert(comment)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultVO{}
	}
	err = session.Commit()
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultVO{}
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
	}
	var comments []entity.TComment
	for _, v := range vo.Ids {
		comment := entity.TComment{
			Id:       v,
			IsReview: vo.IsReview,
		}
		comments = append(comments, comment)
	}
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	for _, v := range comments {
		_, err = session.Prepare().ID(v.Id).MustCols("is_review").Update(&v)
		if err != nil {
			zlog.Error(err.Error())
			zlog.Unwrap(session.Rollback())
			return model.ResultFail()
		}
	}
	session.Commit()
	return model.ResultOk()
}

func (c *MyCommentService) DeleteComments(ctx *gin.Context) model.ResultVO {
	var iDs []int
	err := ctx.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
	}
	_, err = ormInit.GetEngine().In("id", iDs).Delete(&entity.TComment{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (c *MyCommentService) checkComment(vo model.CommentVO) string {
	engine := ormInit.GetEngine()
	if len(shared.TypeHM[vo.Type]) == 0 {
		return "参数校验异常"
	}

	if vo.Type == shared.ARTICLE || vo.Type == shared.TALK {
		if vo.TopicId == "" {
			return "参数校验异常"
		} else {
			if vo.Type == shared.ARTICLE {
				var article entity.TArticle
				get, err := engine.SQL("select id, user_id from t_article where id = " + vo.TopicId).Get(&article)
				if err != nil || !get {
					return "参数校验异常"
				}
			}
			if vo.Type == shared.TALK {
				var talk entity.TTalk
				get, err := engine.SQL("select id, user_id from t_talk where id = " + vo.TopicId).Get(&talk)
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
		get, err := engine.SQL("select id, parent_id, type from t_comment where id = " + strconv.Itoa(vo.ParentId)).Get(&parentComment)
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
			get, err := engine.SQL("select id from t_user_info where id = " + strconv.Itoa(vo.ReplyUserId)).Get(&userInfo)
			if err != nil || !get {
				return "参数校验异常"
			}
		}
	}
	return ""
}
