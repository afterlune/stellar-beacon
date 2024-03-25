package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/SearchEngines"
	"benetnasch/app/infrastructure/oss"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/shared"
	"benetnasch/app/infrastructure/zlog"
	"bytes"
	"container/list"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"xorm.io/builder"
	"xorm.io/xorm"
)

func ListTopAndFeaturedArticles() model.ResultVO {
	data := repository.ListTopAndFeaturedArticles()
	if len(data) == 0 {
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{})
	} else if len(data) > 3 {
		data = data[:3]
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{TopArticle: data[0], FeaturedArticles: data[1:3]})
	} else {
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{TopArticle: data[0], FeaturedArticles: data[1:]})
	}
}

func ListArticles(c *gin.Context) model.ResultVO {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	var count int
	_, err := ormInit.GetEngine().SQL("SELECT count(0) from t_article where is_delete =0 and status = 1").Get(&count)
	if err != nil {
		zlog.Error(err.Error())
	}
	data := repository.ListArticles(current, size)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: count})
}

func ListArticlesByCategoryId(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		zlog.Error(err.Error())
	}

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		zlog.Error(err.Error())
	}

	categoryId, _ := strconv.Atoi(c.Query("categoryId"))
	var count int
	_, err = ormInit.GetEngine().SQL("select count(0) from t_article where category_id = " + strconv.Itoa(categoryId)).Get(&count)
	if err != nil {
		zlog.Error(err.Error())
	}

	data := repository.GetArticlesByCategoryId(current, size, categoryId)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: count})
}

func GetArticleById(c *gin.Context) model.ResultVO {
	articleId := c.Param("articleId")
	get := shared.Get(articleId)
	if get != "" {
		var dto model.ArticleDTO
		shared.Unmarsh(get, &dto)
		shared.Expire(articleId, time.Hour*1)
		return model.ResultOkWithData(dto)
	}
	var article entity.TArticle
	_, err := ormInit.GetEngine().ID(articleId).Get(&article)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultVO{}
	}
	if &article == nil {
		return model.ResultOk()
	}
	if article.Status == 2 {
		var isAccess bool
		value, _ := c.Get("userInfo")
		dto := value.(model.UserDetailsDTO)
		isAccess = shared.SIsMember(shared.ARTICLE_ACCESS+strconv.Itoa(dto.Id), articleId)
		if isAccess == false {
			status := model.ResultInfo(model.ARTICLE_ACCESS_FAIL)
			return model.ResultFailWithCodeAndMessage(52003, status["message"])
		}
	}
	updateArticleViewsCount(articleId)
	id, err := strconv.Atoi(articleId)
	if err != nil {
		zlog.Error(err.Error())
	}

	data := repository.GetArticleById(id)
	preData := repository.GetPreArticleById(id)
	if preData.Id == 0 {
		preData = repository.GetLastArticle()
	}
	nextData := repository.GetNextArticleById(id)
	if nextData.Id == 0 {
		nextData = repository.GetFirstArticle()
	}
	if data.Id == 0 {
		return model.ResultOk()
	}
	score := shared.ZScore(shared.ARTICLE_VIEWS_COUNT, articleId)
	if score == 0 {
		data.ViewCount = int(score)
	}
	data.PreArticleCard = preData
	data.NextArticleCard = nextData
	marshal, _ := json.Marshal(data)
	shared.SetWithTime(strconv.Itoa(data.Id), marshal, time.Hour*1)
	return model.ResultOkWithData(data)
}

func updateArticleViewsCount(articleId string) {
	shared.ZIncr(shared.ARTICLE_VIEWS_COUNT, 1, articleId)
}

func ListArticlesByTagId(c *gin.Context) model.ResultVO {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	tagId := c.Query("tagId")
	count, err := ormInit.GetEngine().ID(tagId).Count(&entity.TTag{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	id, _ := strconv.Atoi(tagId)
	data := repository.ListArticlesByTagId(current, size, id)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func AccessArticle(c *gin.Context) model.ResultVO {
	var vo model.ArticlePasswordVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var article entity.TArticle
	_, err = ormInit.GetEngine().Where(builder.Eq{"id": vo.ArticleId}).Get(&article)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if article.Id == 0 {
		return model.ResultFailWithMessage("文章不存在")
	}
	if article.Password == vo.ArticlePassword {
		value, _ := c.Get("userInfo")
		dto := value.(model.UserDetailsDTO)
		shared.SAdd(shared.ARTICLE_ACCESS+strconv.Itoa(dto.Id), vo.ArticleId)
	} else {
		return model.ResultFailWithMessage("密码错误")
	}
	return model.ResultOk()
}

func ListArchives(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		zlog.Error(err.Error())
	}

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		zlog.Error(err.Error())
	}

	count, err := ormInit.GetEngine().Where("is_delete = 0 and status = 1").Count(&entity.TArticle{})
	if err != nil {
		zlog.Error(err.Error())
	}

	articles := repository.ListArchives(current, size)
	hm := make(map[string][]model.ArticleCardDTO)
	for _, v := range articles {
		year, month, day := v.CreateTime.Date()
		key := strconv.Itoa(year) + "-" + strconv.Itoa(int(month)) + "-" + strconv.Itoa(day)
		value := hm[key]
		if value == nil {
			var articleCardDTOS []model.ArticleCardDTO
			articleCardDTOS = append(articleCardDTOS, v)
			hm[key] = articleCardDTOS
		} else {
			articleCards := hm[key]
			articleCards = append(articleCards, v)
			hm[key] = articleCards
		}
	}
	var archiveDTOs []model.ArchiveDTO
	for k, v := range hm {
		var archiveDTO model.ArchiveDTO
		archiveDTO.Time = k
		archiveDTO.Articles = v
		archiveDTOs = append(archiveDTOs, archiveDTO)
	}
	sort.Slice(archiveDTOs, func(i, j int) bool {

		is := strings.Split(archiveDTOs[i].Time, "-")
		js := strings.Split(archiveDTOs[j].Time, "-")
		iyear, err := strconv.Atoi(is[0])
		if err != nil {
			zlog.Error(err.Error())
		}

		imonth, err := strconv.Atoi(is[1])
		if err != nil {
			zlog.Error(err.Error())
		}

		iday, err := strconv.Atoi(is[2])
		if err != nil {
			zlog.Error(err.Error())
		}

		jyear, err := strconv.Atoi(js[0])
		if err != nil {
			zlog.Error(err.Error())
		}

		jmonth, err := strconv.Atoi(js[1])
		if err != nil {
			zlog.Error(err.Error())
		}

		jday, err := strconv.Atoi(js[2])
		if err != nil {
			zlog.Error(err.Error())
		}

		if iyear > jyear {
			return false
		} else if iyear < jyear {
			return true
		}
		if imonth > jmonth {
			return false
		} else if imonth < jmonth {
			return true
		}
		return iday < jday
	})
	if len(archiveDTOs) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: archiveDTOs, Count: int(count)})
}

func ListArticlesAdmin(c *gin.Context) model.ResultVO {
	var conditionVO model.ConditionVO
	err := c.ShouldBindQuery(&conditionVO)
	if err != nil {
		zlog.Error(err.Error())
	}
	count := repository.CountArticleAdmins(&conditionVO)
	articleAdminDTOs := repository.ListArticlesAdmin(conditionVO.Current, conditionVO.Size, &conditionVO)
	viewsCountMap := shared.ZAllScore(shared.ARTICLE_VIEWS_COUNT)
	for _, v := range articleAdminDTOs {
		index := strconv.Itoa(v.Id)
		viewsCount := viewsCountMap[index]
		if viewsCount != 0 {
			v.ViewsCount = int(viewsCount)
		}
	}
	if len(articleAdminDTOs) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: articleAdminDTOs, Count: count})
}

func SaveOrUpdateArticle(c *gin.Context) model.ResultVO {
	var articleVO model.ArticleVO
	err := c.ShouldBind(&articleVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	vo, ok := c.Get("articleVO")
	if ok {
		articleVO = vo.(model.ArticleVO)
	}
	session := ormInit.GetEngine().NewSession()
	defer func(session *xorm.Session) {
		err := session.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(session)
	err = session.Begin()
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	category := saveArticleCategory(articleVO, session)
	var article entity.TArticle
	marshal, err := json.Marshal(articleVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	err = json.Unmarshal(marshal, &article)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if category.Id != 0 {
		article.CategoryId = category.Id
	}

	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)

	article.UserId = dto.UserInfoId
	if article.Id != 0 {
		_, err = session.Prepare().ID(article.Id).Update(&article)
	} else {
		_, err = session.Prepare().Insert(&article)
	}
	if err != nil {
		zlog.Error(err.Error())
		err := session.Rollback()
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFail()
	}
	saveArticleTag(articleVO, article.Id, session)
	if article.Status == 1 {
		// TODO subscribe article; args: article.Id
	}
	var articlebase entity.TArticle
	_, err = ormInit.GetEngine().Prepare().ID(articleVO.Id).Get(&articlebase)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if articlebase.Id != 0 {
		marsha, err := json.Marshal(articlebase)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		get := shared.Get(strconv.Itoa(articlebase.Id))
		if get == "" {
			shared.Set(strconv.Itoa(articlebase.Id), marsha)
		} else {
			shared.Del(strconv.Itoa(articlebase.Id))
			shared.Set(strconv.Itoa(articlebase.Id), marsha)
		}
	}
	err = session.Commit()
	if err != nil {
		zlog.Error(err.Error())
		zlog.Unwrap(session.Rollback())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func UpdateArticleTopAndFeatured(c *gin.Context) model.ResultVO {
	var articleTopFeaturedVO model.ArticleTopFeaturedVO
	err := c.ShouldBind(&articleTopFeaturedVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	article := entity.TArticle{
		Id:         articleTopFeaturedVO.Id,
		IsTop:      articleTopFeaturedVO.IsTop,
		IsFeatured: articleTopFeaturedVO.IsFeatured,
	}
	session := ormInit.GetEngine().NewSession()
	err = session.Begin()
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	defer func(session *xorm.Session) {
		err := session.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}(session)
	_, err = session.Exec(fmt.Sprintf("update t_article set is_top = %d, is_featured = %d where id = %d", article.IsTop, article.IsFeatured, article.Id))
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	err = session.Commit()
	if err != nil {
		zlog.Error(err.Error())
		zlog.Unwrap(session.Rollback())
		return model.ResultFail()
	}
	var articlebase entity.TArticle
	_, err = ormInit.GetEngine().Prepare().ID(article.Id).Get(&articlebase)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if articlebase.Id != 0 {
		marsha, err := json.Marshal(articlebase)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		get := shared.Get(strconv.Itoa(articlebase.Id))
		if get == "" {
			shared.Set(strconv.Itoa(articlebase.Id), marsha)
		} else {
			shared.Del(strconv.Itoa(articlebase.Id))
			shared.Set(strconv.Itoa(articlebase.Id), marsha)
		}
	}
	return model.ResultOk()
}

func UpdateArticleDelete(c *gin.Context) model.ResultVO {
	var deleteVO model.DeleteVO
	err := c.ShouldBind(&deleteVO)
	if err != nil {
		zlog.Error(err.Error())
	}
	var articles []entity.TArticle
	for _, v := range deleteVO.Ids {
		articles = append(articles, entity.TArticle{
			Id:       v,
			IsDelete: deleteVO.IsDelete,
		})
	}
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	for _, v := range articles {
		_, err = session.Prepare().MustCols("is_delete").ID(v.Id).Update(&v)
	}
	if err != nil {
		zlog.Error(err.Error())
		zlog.Unwrap(session.Rollback())
		return model.ResultFail()
	}
	err = session.Commit()
	if err != nil {
		zlog.Error(err.Error())
		zlog.Unwrap(session.Rollback())
	}
	return model.ResultOk()
}

func DeleteArticles(c *gin.Context) model.ResultVO {
	var ids []int
	err := c.ShouldBind(&ids)
	if err != nil {
		zlog.Error(err.Error())
	}
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	for _, id := range ids {
		_, err = session.Prepare().Exec("delete from t_article where id = " + strconv.Itoa(id))
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		_, err = session.Where(builder.Eq{"article_id": id}).Delete(&entity.TArticleTag{})
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
	}
	err = session.Commit()
	if err != nil {
		err = session.Rollback()
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func SaveArticleImages(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	fileUri := oss.Upload(file, "articles/")
	return model.ResultOkWithData(shared.FILEURL + fileUri)
}

func GetArticleBackById(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("articleId"))
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var article entity.TArticle
	_, err = engine.ID(id).Get(&article)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var category entity.TCategory
	_, err = engine.ID(article.CategoryId).Get(&category)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	categoryName := ""
	if category.Id != 0 {
		categoryName = category.CategoryName
	}
	tagNames := repository.ListTagNamesByArticleId(id)
	var articleAdminViewDTO model.ArticleAdminViewDTO
	marshal, err := json.Marshal(article)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	err = json.Unmarshal(marshal, &articleAdminViewDTO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	articleAdminViewDTO.CategoryName = categoryName
	if len(tagNames) == 0 {
		articleAdminViewDTO.TagNames = list.New()
	} else {
		articleAdminViewDTO.TagNames = tagNames
	}
	return model.ResultOkWithData(articleAdminViewDTO)
}

func ImportArticles(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	filename := file.Filename
	index := strings.LastIndex(filename, ".")
	articleTitle := filename[:index]
	content, err := file.Open()
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	all, err := io.ReadAll(content)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	articleVO := model.ArticleVO{
		ArticleTitle:   articleTitle,
		ArticleContent: string(all),
		Status:         3,
	}
	c.Set("articleVO", articleVO)
	SaveOrUpdateArticle(c)
	return model.ResultOk()
}

func ExportArticles(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage("导出文章失败")
	}
	var articles []entity.TArticle
	err = ormInit.GetEngine().Select("article_title, article_content").In("id", iDs).Find(&articles)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage("导出文章失败")
	}
	var urls []string
	for _, v := range articles {
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFailWithMessage("导出文章失败")
		}

		fileUri := oss.UploadFile(bytes.NewReader([]byte(v.ArticleContent)), v.ArticleTitle+".md", "markdown/")
		urls = append(urls, shared.FILEURL+fileUri)
	}
	return model.ResultOkWithData(urls)
}

func ListArticlesBySearch(c *gin.Context) model.ResultVO {
	keywords := c.Query("keywords")
	if keywords == "" {
		return model.ResultOk()
	}
	data := SearchEngines.Search(keywords)
	for _, v := range data {
		v1 := v.(map[string]interface{})
		v1["articleTitle"] = v1["_formatted"].(map[string]interface{})["articleTitle"]
		v1["articleContent"] = v1["_formatted"].(map[string]interface{})["articleContent"]
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	var articleSearchDTOs []model.ArticleSearchDTO
	err = json.Unmarshal(jsonData, &articleSearchDTOs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	return model.ResultOkWithData(articleSearchDTOs)
}

func saveArticleCategory(vo model.ArticleVO, session *xorm.Session) entity.TCategory {
	var category entity.TCategory
	_, err := session.Prepare().SQL("select * from t_category where category_name = '" + vo.CategoryName + "'").Get(&category)
	if err != nil {
		zlog.Error(err.Error())
		return entity.TCategory{}
	}
	if category.Id == 0 && vo.Status != 3 {
		category.CategoryName = vo.CategoryName
		_, err = session.Prepare().Insert(&category)
		if err != nil {
			zlog.Error(err.Error())
			zlog.Unwrap(session.Rollback())
			return entity.TCategory{}
		}
	}
	return category
}

func saveArticleTag(vo model.ArticleVO, articleId int, session *xorm.Session) {
	var atag entity.TArticleTag
	if vo.Id != 0 {
		_, err := session.Prepare().Where("article_id = " + strconv.Itoa(vo.Id)).Delete(&atag)
		if err != nil {
			zlog.Error(err.Error())
			zlog.Unwrap(session.Rollback())
			return
		}
	}
	tagNames := vo.TagNames
	if len(tagNames) != 0 {
		var existTags []entity.TTag
		err := session.Prepare().In("tag_name", tagNames).Find(&existTags)
		if err != nil {
			return
		}
		var existTagIds []int
		for _, v := range existTags {
			for k, value := range tagNames {
				if v.TagName == value {
					tagNames = append(tagNames[:k], tagNames[k+1:]...)
				}
			}
			existTagIds = append(existTagIds, v.Id)
		}
		var tags []entity.TTag
		if len(tagNames) != 0 {
			for _, v := range tagNames {
				tags = append(tags, entity.TTag{TagName: v})
			}
			for k, tag := range tags {
				_, err := session.Prepare().Insert(&tag)
				if err != nil {
					zlog.Error(err.Error())
					return
				}
				tags[k] = tag
			}

			var tagIds []int
			for _, tag := range tags {
				tagIds = append(tagIds, tag.Id)
			}
			existTagIds = append(existTagIds, tagIds...)
		}
		var articleTags []entity.TArticleTag
		for _, tagId := range existTagIds {
			articleTags = append(articleTags, entity.TArticleTag{
				ArticleId: articleId,
				TagId:     tagId,
			})
		}
		_, err = session.Prepare().Insert(&articleTags)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
	}
}
