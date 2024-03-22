package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"xorm.io/builder"
)

func ListTags() model.ResultVO {
	data := repository.ListTags()
	return model.ResultOkWithData(data)
}

func ListTopTenTags() model.ResultVO {
	data := repository.ListTopTenTags()
	return model.ResultOkWithData(data)
}

func ListTagsAdmin(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	var count int64
	if vo.Keywords != "" {
		count, err = ormInit.GetEngine().Where("tag_name", vo.Keywords).Count(&entity.TTag{})
	} else {
		count, err = ormInit.GetEngine().Count(&entity.TTag{})
	}
	if err != nil {
		zlog.Error(err.Error())
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: repository.ListTagsAdmin(vo.Current, vo.Size, &vo), Count: int(count)})
}

func ListTagsAdminBySearch(c *gin.Context) model.ResultVO {
	var conditionVO model.ConditionVO
	err := c.ShouldBind(&conditionVO)
	if err != nil {
		zlog.Error(err.Error())
	}
	var tags []entity.TTag
	err = ormInit.GetEngine().Where(builder.Like{"tag_name", conditionVO.Keywords}).OrderBy("id").Desc("id").Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	marshal, err := json.Marshal(tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	var tagAdminDTOs []model.TagAdminDTO
	err = json.Unmarshal(marshal, &tagAdminDTOs)
	if err != nil {
		zlog.Error(err.Error())
	}
	return model.ResultOkWithData(tagAdminDTOs)
}

func SaveOrUpdateTag(c *gin.Context) model.ResultVO {
	var vo model.TagVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	var tag entity.TTag
	engine := ormInit.GetEngine()
	_, err = engine.Select("id").Where("tag_name = '" + vo.TagName + "'").Get(&tag)
	if err != nil {
		zlog.Error(err.Error())
	}
	if tag.Id != 0 && tag.Id != vo.Id {
		return model.ResultFailWithMessage("标签名已存在")
	}
	tag.TagName = vo.TagName
	session := engine.NewSession()
	session.Begin()
	defer session.Close()
	if tag.Id != 0 {
		_, err = session.ID(tag.Id).Update(&tag)
	} else {
		_, err = session.Insert(&tag)
	}
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	session.Commit()
	return model.ResultOk()
}

func DeleteTag(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
	}
	engine := ormInit.GetEngine()
	count, err := engine.In("tag_id", iDs).Count(&entity.TArticleTag{})
	if err != nil {
		zlog.Error(err.Error())
	}
	if count > 0 {
		return model.ResultFailWithMessage("删除失败，该标签下存在文章")
	}
	_, err = engine.In("id", iDs).Delete(&entity.TTag{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}
