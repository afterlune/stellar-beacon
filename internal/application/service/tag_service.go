package service

import (
	"container/list"
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type TagService interface {
	ListTags() model.ResultVO
	ListTopTenTags() model.ResultVO
	ListTagsAdmin(c *gin.Context) model.ResultVO
	ListTagsAdminBySearch(c *gin.Context) model.ResultVO
	SaveOrUpdateTag(c *gin.Context) model.ResultVO
	DeleteTag(c *gin.Context) model.ResultVO
}

type MyTagService struct {
	repo port.TagRepository
}

func NewTagService(repo port.TagRepository) *MyTagService {
	return &MyTagService{repo: repo}
}

func (t *MyTagService) tagRepository() port.TagRepository {
	if t.repo != nil {
		return t.repo
	}
	return tagRepo
}

func (t *MyTagService) ListTags() model.ResultVO {
	data, err := t.tagRepository().List(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(data)
}

func (t *MyTagService) ListTopTenTags() model.ResultVO {
	data, err := t.tagRepository().ListTopTen(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(data)
}

func (t *MyTagService) ListTagsAdmin(ctx *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.TagFilter{Keywords: vo.Keywords}
	count, err := t.tagRepository().CountAdmin(ctx.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	data, err := t.tagRepository().ListAdmin(ctx.Request.Context(), vo.Current, vo.Size, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (t *MyTagService) ListTagsAdminBySearch(ctx *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	data, err := t.tagRepository().Search(ctx.Request.Context(), vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(data)
}

func (t *MyTagService) SaveOrUpdateTag(ctx *gin.Context) model.ResultVO {
	var vo model.TagVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	value, ok := ctx.Get("userInfo")
	if !ok {
		return model.ResultFailWithMessage("用户未登录")
	}
	user, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFailWithMessage("用户信息无效")
	}
	tag := entity.TTag{Id: vo.Id, UserId: user.UserInfoId, TagName: vo.TagName}
	if err := t.tagRepository().SaveOrUpdate(ctx.Request.Context(), tag); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return model.ResultFailWithMessage("标签名已存在")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (t *MyTagService) DeleteTag(ctx *gin.Context) model.ResultVO {
	var ids []int
	if err := ctx.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	if err := t.tagRepository().Delete(ctx.Request.Context(), user.UserInfoId, ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return model.ResultFailWithMessage("删除失败，该标签下存在文章")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

var _ TagService = (*MyTagService)(nil)
