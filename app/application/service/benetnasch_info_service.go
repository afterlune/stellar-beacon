package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"net/http"
	"sort"
	"strconv"
)

type BenetnaschInfoService interface {
	GetBenetnaschHomeInfo() model.ResultVO
	Report(req *http.Request) model.ResultVO
	GetBlogHomeInfo() model.ResultVO
	GetWebsiteConfig() model.ResultVO
	GetBlogBackInfo() model.ResultVO
	UpdateWebsiteConfig(c *gin.Context) model.ResultVO
	GetAbout() model.ResultVO
	UpdateAbout(c *gin.Context) model.ResultVO
	SaveBlogPhotoAlbumCover(c *gin.Context) model.ResultVO
	listArticleRank(hm map[interface{}]float64) []model.ArticleRankDTO
}

type MyBenetnaschInfoService struct{}

func (b *MyBenetnaschInfoService) GetBenetnaschHomeInfo() model.ResultVO {
	return model.ResultVO{}
}

func (b *MyBenetnaschInfoService) Report(req *http.Request) model.ResultVO {
	md5 := shared.GetMD5(shared.GetRedisId(req))
	if !shared.SIsMember(shared.UNIQUE_VISITOR, md5) {
		ipSource := shared.GetIpSource(shared.GetIpAddress(req))
		if ipSource != "" {
			shared.HIncrBy(shared.VISITOR_AREA, ipSource, 1)
		} else {
			shared.HIncrBy(shared.VISITOR_AREA, shared.UNKNOWN, 1)
		}
		shared.IncrBy(shared.BLOG_VIEWS_COUNT, 1)
		shared.SAdd(shared.UNIQUE_VISITOR, md5)
	}
	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) GetBlogHomeInfo() model.ResultVO {
	var articleCount, categoryCount, tagCount, talkCount int64
	engine := ormInit.GetEngine()
	_, err := engine.SQL("select count(0) from t_article where is_delete = 0").Get(&articleCount)
	if err != nil {
		zlog.Error(err.Error())
	}

	_, err = engine.SQL("select count(0) from t_category").Get(&categoryCount)
	if err != nil {
		zlog.Error(err.Error())
	}

	_, err = engine.SQL("select count(0) from t_tag").Get(&tagCount)
	if err != nil {
		zlog.Error(err.Error())
	}

	_, err = engine.SQL("select count(0) from t_talk").Get(&talkCount)
	if err != nil {
		zlog.Error(err.Error())
	}

	reData := shared.Get(shared.BLOG_VIEWS_COUNT).(string)
	var viewCount int
	if reData != "" {
		viewCount, _ = strconv.Atoi(reData)
	} else {
		viewCount = 0
	}
	websiteConfig := b.GetWebsiteConfig().Data.(model.WebsiteConfigDTO)
	return model.ResultOkWithData(model.BenetnaschHomeInfoDTO{
		ArticleCount:    articleCount,
		CategoryCount:   categoryCount,
		TagCount:        tagCount,
		TalkCount:       talkCount,
		ViewCount:       viewCount,
		WebsiteConfigDT: websiteConfig,
	})
}

func (b *MyBenetnaschInfoService) GetWebsiteConfig() model.ResultVO {
	var webConfig model.WebsiteConfigDTO
	var config string
	websiteConfig := shared.Get(shared.WEBSITE_CONFIG).(string)
	if websiteConfig != "" {
		zlog.Unwrap(json.Unmarshal([]byte(websiteConfig), &webConfig))
	} else {
		_, err := ormInit.GetEngine().SQL("select config from t_website_config where id = 1").Get(&config)
		if err != nil {
			zlog.Error(err.Error())
		}

		zlog.Unwrap(json.Unmarshal([]byte(config), &webConfig))

		shared.Set(shared.WEBSITE_CONFIG, config)
	}
	return model.ResultOkWithData(webConfig)
}

func (b *MyBenetnaschInfoService) GetBlogBackInfo() model.ResultVO {
	count, err := strconv.Atoi(shared.Get(shared.BLOG_VIEWS_COUNT).(string))
	if err != nil {
		zlog.Error(err.Error())
	}
	engine := ormInit.GetEngine()
	messageCount, err := engine.Where("type = 2").Count(&entity.TComment{})
	if err != nil {
		zlog.Error(err.Error())
	}
	userCount, err := engine.Count(&entity.TUserInfo{})
	if err != nil {
		zlog.Error(err.Error())
	}
	articleCount, err := engine.Where("is_delete = 0").Count(&entity.TArticle{})
	if err != nil {
		zlog.Error(err.Error())
	}
	uniqueViews := listUniqueViews()
	articleStatisticsDTOs := articleRepo.ListArticleStatistics()
	categoryDTOs := categoryRepo.ListCategories()
	var tags []entity.TTag
	err = engine.Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	marshal, err := json.Marshal(tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	var tagDTOs []model.TagDTO
	err = json.Unmarshal(marshal, &tagDTOs)
	if err != nil {
		zlog.Error(err.Error())
	}
	articleMap := shared.ZReverseRangeWithScore(shared.ARTICLE_VIEWS_COUNT, 0, 4)
	auroraAdminInfoDTO := model.BenetnaschBackInfoDTO{
		ArticleStatisticsDTOs: articleStatisticsDTOs,
		TagDTOs:               tagDTOs,
		ViewsCount:            count,
		MessageCount:          int(messageCount),
		UserCount:             int(userCount),
		ArticleCount:          int(articleCount),
		CategoryDTOs:          categoryDTOs,
		UniqueViewDTOs:        uniqueViews,
	}
	if len(articleMap) != 0 {
		articleRankDTOs := b.listArticleRank(articleMap)
		auroraAdminInfoDTO.ArticleRankDTOs = articleRankDTOs
	}
	return model.ResultOkWithData(auroraAdminInfoDTO)
}

func (b *MyBenetnaschInfoService) UpdateWebsiteConfig(c *gin.Context) model.ResultVO {
	var webCfg model.WebsiteConfigVO
	if err := c.ShouldBind(&webCfg); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	m, err := json.Marshal(&webCfg)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	engine := ormInit.GetEngine()
	session := engine.NewSession()
	defer session.Close()
	session.Begin()
	_, err = session.Exec("update t_website_config set config = ? where id = 1", string(m))
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFailWithMessage(err.Error())
	}
	session.Commit()
	shared.Set(shared.WEBSITE_CONFIG, string(m))

	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) GetAbout() model.ResultVO {
	var aboutDTO model.AboutDTO
	about := shared.Get(shared.ABOUT).(string)
	if about != "" {
		err := json.Unmarshal([]byte(about), &aboutDTO)
		if err != nil {
			zlog.Error(err.Error())
		}
	} else {
		var abt entity.TAbout
		_, err := ormInit.GetEngine().ID(shared.DEFAULT_ABOUT_ID).Get(&abt)
		if err != nil {
			zlog.Error(err.Error())
		}

		err = json.Unmarshal([]byte(abt.Content), &aboutDTO)
		if err != nil {
			zlog.Error(err.Error())
		}

		shared.Set(shared.ABOUT, abt.Content)
	}
	return model.ResultOkWithData(aboutDTO)
}

func (b *MyBenetnaschInfoService) UpdateAbout(c *gin.Context) model.ResultVO {
	var abt model.AboutVO
	if err := c.ShouldBind(&abt); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	engine := ormInit.GetEngine()
	session := engine.NewSession()
	defer session.Close()
	session.Begin()
	_, err := session.Exec("update t_about set config = ? where id = 1", abt.Content)
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFailWithMessage(err.Error())
	}
	session.Commit()
	shared.Set(shared.ABOUT, abt.Content)

	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) SaveBlogPhotoAlbumCover(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) listArticleRank(hm map[interface{}]float64) []model.ArticleRankDTO {
	var articleIds []int
	for k, _ := range hm {
		id, err := strconv.Atoi(k.(string))
		if err != nil {
			zlog.Error(err.Error())
		}
		articleIds = append(articleIds, id)
	}
	var articles []entity.TArticle
	err := ormInit.GetEngine().Select("id, article_title").In("id", articleIds).Find(&articles)
	if err != nil {
		zlog.Error(err.Error())
	}
	var articleRankDTOs []model.ArticleRankDTO
	for _, v := range articles {
		articleRankDTO := model.ArticleRankDTO{
			ArticleTitle: v.ArticleTitle,
			ViewsCount:   int(hm[strconv.Itoa(v.Id)]),
		}
		articleRankDTOs = append(articleRankDTOs, articleRankDTO)
	}
	sort.Slice(articleRankDTOs, func(i, j int) bool {
		return articleRankDTOs[i].ViewsCount > articleRankDTOs[j].ViewsCount
	})
	return articleRankDTOs
}
