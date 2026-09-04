package service

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
)

type TagService interface {
	ListTags(ctx context.Context) port.ResultVO
	ListTopTenTags(ctx context.Context) port.ResultVO
	ListTagsAdmin(c port.Request) port.ResultVO
	ListTagsAdminBySearch(c port.Request) port.ResultVO
	SaveOrUpdateTag(c port.Request) port.ResultVO
	DeleteTag(c port.Request) port.ResultVO
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

func (t *MyTagService) ListTags(ctx context.Context) port.ResultVO {
	data, err := t.tagRepository().List(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(data)
}

func (t *MyTagService) ListTopTenTags(ctx context.Context) port.ResultVO {
	data, err := t.tagRepository().ListTopTen(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(data)
}

func (t *MyTagService) ListTagsAdmin(ctx port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := ctx.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.TagFilter{Keywords: vo.Keywords}
	count, err := t.tagRepository().CountAdmin(ctx.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	data, err := t.tagRepository().ListAdmin(ctx.Context(), vo.Current, vo.Size, filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: data, Count: int(count)})
}

func (t *MyTagService) ListTagsAdminBySearch(ctx port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := ctx.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	data, err := t.tagRepository().Search(ctx.Context(), vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(data)
}

func (t *MyTagService) SaveOrUpdateTag(ctx port.Request) port.ResultVO {
	var vo port.TagVO
	if err := ctx.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	tag := port.TTag{Id: vo.Id, TagName: vo.TagName}
	if err := t.tagRepository().SaveOrUpdate(ctx.Context(), tag); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return port.ResultFailWithMessage("标签名已存在")
		}
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (t *MyTagService) DeleteTag(ctx port.Request) port.ResultVO {
	var ids []int
	if err := ctx.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := t.tagRepository().Delete(ctx.Context(), ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return port.ResultFailWithMessage("删除失败，该标签下存在文章")
		}
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

var _ TagService = (*MyTagService)(nil)
