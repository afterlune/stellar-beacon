package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/SearchEngines"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"bytes"
	"container/list"
	"context"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type ArticleService interface {
	ListTopAndFeaturedArticles() model.ResultVO
	ListArticles(c *gin.Context) model.ResultVO
	ListArticlesByCategoryId(c *gin.Context) model.ResultVO
	GetArticleById(c *gin.Context) model.ResultVO
	updateArticleViewsCount(ctx context.Context, articleId string)
	ListArticlesByTagId(c *gin.Context) model.ResultVO
	AccessArticle(c *gin.Context) model.ResultVO
	ListArchives(c *gin.Context) model.ResultVO
	ListArticlesAdmin(c *gin.Context) model.ResultVO
	SaveOrUpdateArticle(c *gin.Context) model.ResultVO
	UpdateArticleTopAndFeatured(c *gin.Context) model.ResultVO
	UpdateArticleDelete(c *gin.Context) model.ResultVO
	DeleteArticles(c *gin.Context) model.ResultVO
	SaveArticleImages(c *gin.Context) model.ResultVO
	GetArticleBackById(c *gin.Context) model.ResultVO
	ImportArticles(c *gin.Context) model.ResultVO
	ExportArticles(c *gin.Context) model.ResultVO
	ListArticlesBySearch(c *gin.Context) model.ResultVO
	saveArticleCategory(vo model.ArticleVO, session *xorm.Session) (entity.TCategory, error)
	saveArticleTag(vo model.ArticleVO, articleId int, session *xorm.Session) error
}

type MyArticleService struct{}

func (a *MyArticleService) ListTopAndFeaturedArticles() model.ResultVO {
	data := articleRepo.ListTopAndFeaturedArticles()
	if len(data) == 0 {
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{})
	} else if len(data) > 3 {
		data = data[:3]
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{TopArticle: data[0], FeaturedArticles: data[1:3]})
	} else {
		return model.ResultOkWithData(model.TopAndFeaturedArticlesDTO{TopArticle: data[0], FeaturedArticles: data[1:]})
	}
}

func (a *MyArticleService) ListArticles(c *gin.Context) model.ResultVO {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	var count int
	_, err := ormInit.GetEngine().SQL("SELECT count(0) from t_article where is_delete =0 and status = 1").Get(&count)
	if err != nil {
		zlog.Error(err.Error())
	}
	data := articleRepo.ListArticles(current, size)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: count})
}

func (a *MyArticleService) ListArticlesByCategoryId(c *gin.Context) model.ResultVO {
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
	_, err = ormInit.GetEngine().SQL("select count(0) from t_article where category_id = ?", categoryId).Get(&count)
	if err != nil {
		zlog.Error(err.Error())
	}

	data := articleRepo.GetArticlesByCategoryId(current, size, categoryId)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: count})
}

func (a *MyArticleService) GetArticleById(c *gin.Context) model.ResultVO {
	articleId := c.Param("articleId")
	get, err := shared.GetCtx(c.Request.Context(), articleId)
	if err != nil {
		zlog.Error(err.Error())
	}
	if get != "" {
		var dto model.ArticleDTO
		shared.Unmarsh(get, &dto)
		if _, err := shared.ExpireCtx(c.Request.Context(), articleId, time.Hour*1); err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultOkWithData(dto)
	}
	var article entity.TArticle
	_, err = ormInit.GetEngine().Context(c.Request.Context()).ID(articleId).Get(&article)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultVO{}
	}
	if article.Id == 0 {
		return model.ResultOk()
	}
	if article.Status == 2 {
		value, ok := c.Get("userInfo")
		if !ok {
			return model.ResultFailWithMessage("无权访问")
		}
		dto, ok := value.(model.UserDetailsDTO)
		if !ok {
			return model.ResultFailWithMessage("无权访问")
		}
		isAccess, err := shared.SIsMemberCtx(c.Request.Context(), shared.ARTICLE_ACCESS+strconv.Itoa(dto.Id), articleId)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		if !isAccess {
			status := model.ResultInfo(model.ARTICLE_ACCESS_FAIL)
			return model.ResultFailWithCodeAndMessage(52003, status["message"])
		}
	}
	a.updateArticleViewsCount(c.Request.Context(), articleId)
	id, err := strconv.Atoi(articleId)
	if err != nil {
		zlog.Error(err.Error())
	}

	data := articleRepo.GetArticleById(id)
	preData := articleRepo.GetPreArticleById(id)
	if preData.Id == 0 {
		preData = articleRepo.GetLastArticle()
	}
	nextData := articleRepo.GetNextArticleById(id)
	if nextData.Id == 0 {
		nextData = articleRepo.GetFirstArticle()
	}
	if data.Id == 0 {
		return model.ResultOk()
	}
	score, err := shared.ZScoreCtx(c.Request.Context(), shared.ARTICLE_VIEWS_COUNT, articleId)
	if err != nil {
		zlog.Error(err.Error())
	}
	if score == 0 {
		data.ViewCount = int(score)
	}
	data.PreArticleCard = preData
	data.NextArticleCard = nextData
	marshal, _ := json.Marshal(data)
	if err := shared.SetWithTimeCtx(c.Request.Context(), strconv.Itoa(data.Id), marshal, time.Hour*1); err != nil {
		zlog.Error(err.Error())
	}
	return model.ResultOkWithData(data)
}

func (a *MyArticleService) updateArticleViewsCount(ctx context.Context, articleId string) {
	if _, err := shared.ZIncrCtx(ctx, shared.ARTICLE_VIEWS_COUNT, 1, articleId); err != nil {
		zlog.Error(err.Error())
	}
}

func (a *MyArticleService) ListArticlesByTagId(c *gin.Context) model.ResultVO {
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	tagId := c.Query("tagId")
	count, err := ormInit.GetEngine().ID(tagId).Count(&entity.TTag{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	id, _ := strconv.Atoi(tagId)
	data := articleRepo.ListArticlesByTagId(current, size, id)
	if len(data) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (a *MyArticleService) AccessArticle(c *gin.Context) model.ResultVO {
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
		if _, err := shared.SAddCtx(c.Request.Context(), shared.ARTICLE_ACCESS+strconv.Itoa(dto.Id), vo.ArticleId); err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
	} else {
		return model.ResultFailWithMessage("密码错误")
	}
	return model.ResultOk()
}

func (a *MyArticleService) ListArchives(c *gin.Context) model.ResultVO {
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

	articles := articleRepo.ListArchives(current, size)
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

func (a *MyArticleService) ListArticlesAdmin(c *gin.Context) model.ResultVO {
	var conditionVO model.ConditionVO
	err := c.ShouldBindQuery(&conditionVO)
	if err != nil {
		zlog.Error(err.Error())
	}
	count := articleRepo.CountArticleAdmins(&conditionVO)
	articleAdminDTOs := articleRepo.ListArticlesAdmin(conditionVO.Current, conditionVO.Size, &conditionVO)
	viewsCountMap, err := shared.ZAllScoreCtx(c.Request.Context(), shared.ARTICLE_VIEWS_COUNT)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
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

func (a *MyArticleService) SaveOrUpdateArticle(c *gin.Context) model.ResultVO {
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
	value, _ := c.Get("userInfo")
	dto := value.(model.UserDetailsDTO)
	var articlebase entity.TArticle
	err = ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		category, err := a.saveArticleCategory(articleVO, session)
		if err != nil {
			return err
		}
		var article entity.TArticle
		marshal, err := json.Marshal(articleVO)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(marshal, &article); err != nil {
			return err
		}
		if category.Id != 0 {
			article.CategoryId = category.Id
		}
		article.UserId = dto.UserInfoId
		if article.Id != 0 {
			if _, err = session.ID(article.Id).Update(&article); err != nil {
				return err
			}
		} else if _, err = session.Insert(&article); err != nil {
			return err
		}
		if err = a.saveArticleTag(articleVO, article.Id, session); err != nil {
			return err
		}
		_, err = session.ID(article.Id).Get(&articlebase)
		return err
	})
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
		if err := shared.SetCtx(c.Request.Context(), strconv.Itoa(articlebase.Id), marsha); err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
	}
	return model.ResultOk()
}

func (a *MyArticleService) UpdateArticleTopAndFeatured(c *gin.Context) model.ResultVO {
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
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Exec("update t_article set is_top = ?, is_featured = ? where id = ?", article.IsTop, article.IsFeatured, article.Id)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var articlebase entity.TArticle
	_, err = ormInit.GetEngine().Context(c.Request.Context()).ID(article.Id).Get(&articlebase)
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
		if err := shared.SetCtx(c.Request.Context(), strconv.Itoa(articlebase.Id), marsha); err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
	}
	return model.ResultOk()
}

func (a *MyArticleService) UpdateArticleDelete(c *gin.Context) model.ResultVO {
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
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		for _, v := range articles {
			if _, err := session.MustCols("is_delete").ID(v.Id).Update(&v); err != nil {
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

func (a *MyArticleService) DeleteArticles(c *gin.Context) model.ResultVO {
	var ids []int
	err := c.ShouldBind(&ids)
	if err != nil {
		zlog.Error(err.Error())
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		for _, id := range ids {
			if _, err := session.Exec("delete from t_article where id = ?", id); err != nil {
				return err
			}
			if _, err := session.Where(builder.Eq{"article_id": id}).Delete(&entity.TArticleTag{}); err != nil {
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

func (a *MyArticleService) SaveArticleImages(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	fileUri := oss.Upload(file, "articles/")
	return model.ResultOkWithData(shared.FILEURL + fileUri)
}

func (a *MyArticleService) GetArticleBackById(c *gin.Context) model.ResultVO {
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
	tagNames := tagRepo.ListTagNamesByArticleId(id)
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

func (a *MyArticleService) ImportArticles(c *gin.Context) model.ResultVO {
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
	a.SaveOrUpdateArticle(c)
	return model.ResultOk()
}

func (a *MyArticleService) ExportArticles(c *gin.Context) model.ResultVO {
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

func (a *MyArticleService) ListArticlesBySearch(c *gin.Context) model.ResultVO {
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

func (a *MyArticleService) saveArticleCategory(vo model.ArticleVO, session *xorm.Session) (entity.TCategory, error) {
	var category entity.TCategory
	_, err := session.SQL("select * from t_category where category_name = ?", vo.CategoryName).Get(&category)
	if err != nil {
		return entity.TCategory{}, err
	}
	if category.Id == 0 && vo.Status != 3 {
		category.CategoryName = vo.CategoryName
		_, err = session.Prepare().Insert(&category)
		if err != nil {
			return entity.TCategory{}, err
		}
	}
	return category, nil
}

func (a *MyArticleService) saveArticleTag(vo model.ArticleVO, articleId int, session *xorm.Session) error {
	var atag entity.TArticleTag
	if vo.Id != 0 {
		_, err := session.Where("article_id = ?", vo.Id).Delete(&atag)
		if err != nil {
			return err
		}
	}
	if len(vo.TagNames) != 0 {
		var existTags []entity.TTag
		err := session.In("tag_name", vo.TagNames).Find(&existTags)
		if err != nil {
			return err
		}
		existingNames := make(map[string]struct{}, len(existTags))
		var existTagIds []int
		for _, v := range existTags {
			existingNames[v.TagName] = struct{}{}
			existTagIds = append(existTagIds, v.Id)
		}
		var tags []entity.TTag
		for _, name := range vo.TagNames {
			if _, exists := existingNames[name]; !exists {
				tags = append(tags, entity.TTag{TagName: name})
				existingNames[name] = struct{}{}
			}
		}
		if len(tags) != 0 {
			for k, tag := range tags {
				_, err := session.Insert(&tag)
				if err != nil {
					return err
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
		if len(articleTags) != 0 {
			_, err = session.Insert(&articleTags)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
