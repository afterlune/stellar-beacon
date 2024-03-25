package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/oss"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/shared"
	"benetnasch/app/infrastructure/zlog"
	"container/list"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"strconv"
)

func ListTalks(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		zlog.Error(err.Error())
	}

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		zlog.Error(err.Error())
	}

	count, err := ormInit.GetEngine().Where("status = 1").Count(&entity.TTalk{})
	if err != nil {
		zlog.Error(err.Error())
	}

	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	talks := repository.ListTalks(current, size)
	var talkIds []int
	for _, v := range talks {
		talkIds = append(talkIds, v.Id)
	}
	comments := repository.ListCommentCountByTypeAndTopicIds(5, talkIds)
	commentCounthm := make(map[int]int)
	for _, v := range comments {
		commentCounthm[v.Id] = v.CommentCount
	}
	for _, v := range talks {
		v.CommentCount = commentCounthm[v.Id]
		if v.Images != "" {
			var s []string
			err := json.Unmarshal([]byte(v.Images), &s)
			if err != nil {
				zlog.Error(err.Error())
			}
			v.Imgs = s
		}
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: int(count)})
}

func GetTalkById(c *gin.Context) model.ResultVO {
	talkId := c.Param("talkId")
	id, err := strconv.Atoi(talkId)
	if err != nil {
		zlog.Error(err.Error())
	}

	talkDTO := repository.GetTalkById(id)
	if talkDTO.Content == "" {
		return model.ResultFailWithMessage("说说不存在")
	}
	if talkDTO.Images != "" {
		var s []string
		err := json.Unmarshal([]byte(talkDTO.Images), &s)
		if err != nil {
			zlog.Error(err.Error())
		}
		talkDTO.Imgs = s
	}
	commentCountDTO := repository.ListCommentCountByTypeAndTopicId(5, id)
	if &commentCountDTO != nil {
		talkDTO.CommentCount = commentCountDTO.CommentCount
	}
	return model.ResultOkWithData(talkDTO)
}

func SaveTalkImages(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
	}
	fileUrl := oss.Upload(file, "talks/")
	return model.ResultOkWithData(shared.FILEURL + fileUrl)
}

func SaveOrUpdateTalk(c *gin.Context) model.ResultVO {
	var vo model.TalkVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	talk := entity.TTalk{
		Id:      vo.Id,
		Content: vo.Content,
		Images:  vo.Images,
		IsTop:   vo.IsTop,
		Status:  vo.Status,
		UserId:  shared.GetUserDetailsDTO().UserInfoId,
	}
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	if talk.Id != 0 {
		_, err = session.Prepare().ID(talk.Id).Update(&talk)
	} else {
		_, err = session.Prepare().Insert(&talk)
	}
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	session.Commit()
	return model.ResultOk()
}

func DeleteTalks(c *gin.Context) model.ResultVO {
	var iDs []string
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
	}
	var talkIds []int
	for _, v := range iDs {
		id, _ := strconv.Atoi(v)
		talkIds = append(talkIds, id)
	}
	_, err = ormInit.GetEngine().In("id", talkIds).Delete(&entity.TTalk{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func ListBackTalks(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	var count int64
	engine := ormInit.GetEngine()
	if vo.Status != 0 {
		count, err = engine.Where(fmt.Sprintf("status = %d", vo.Status)).Count(&entity.TTalk{})
	} else {
		count, err = engine.Count(&entity.TTalk{})
	}
	if err != nil {
		zlog.Error(err.Error())
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	talkDTOs := repository.ListTalksAdmin(vo.Current, vo.Size, &vo)
	for _, v := range talkDTOs {
		if v.Images != "" {
			var imgs []string
			err = json.Unmarshal([]byte(v.Images), &imgs)
			if err != nil {
				zlog.Error(err.Error())
				return model.ResultFail()
			}
			v.Imgs = imgs
		}
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: talkDTOs, Count: int(count)})
}

func GetBackTalkById(c *gin.Context) model.ResultVO {
	talkId := c.Param("talkId")
	id, err := strconv.Atoi(talkId)
	if err != nil {
		zlog.Error(err.Error())
	}
	talkDTO := repository.GetTalkByIdAdmin(id)
	if talkDTO.Images != "" {
		var s []string
		err := json.Unmarshal([]byte(talkDTO.Images), &s)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		talkDTO.Imgs = s
	}
	return model.ResultOkWithData(talkDTO)
}
