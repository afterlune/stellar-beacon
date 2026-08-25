package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"net/http"
	"sort"
	"strconv"
	"xorm.io/xorm"
)

type BenetnaschInfoService interface {
	GetBenetnaschHomeInfo() model.ResultVO
	Report(req *http.Request) model.ResultVO
	GetBlogHomeInfo(ctx context.Context) model.ResultVO
	GetWebsiteConfig(ctx context.Context) model.ResultVO
	GetBlogBackInfo(ctx context.Context) model.ResultVO
	UpdateWebsiteConfig(c *gin.Context) model.ResultVO
	GetAbout(ctx context.Context) model.ResultVO
	UpdateAbout(c *gin.Context) model.ResultVO
	SaveBlogPhotoAlbumCover(c *gin.Context) model.ResultVO
	listArticleRank(ctx context.Context, hm map[interface{}]float64) []model.ArticleRankDTO
}

type MyBenetnaschInfoService struct{}

func (b *MyBenetnaschInfoService) GetBenetnaschHomeInfo() model.ResultVO {
	return model.ResultVO{}
}

func (b *MyBenetnaschInfoService) Report(req *http.Request) model.ResultVO {
	ctx := req.Context()
	md5 := shared.GetMD5(shared.GetRedisId(req))
	seen, err := shared.SIsMemberCtx(ctx, shared.UNIQUE_VISITOR, md5)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if !seen {
		ipSource := shared.GetIpSource(shared.GetIpAddress(req))
		if ipSource != "" {
			if _, err := shared.HIncrByCtx(ctx, shared.VISITOR_AREA, ipSource, 1); err != nil {
				zlog.Error(err.Error())
				return model.ResultFail()
			}
		} else {
			if _, err := shared.HIncrByCtx(ctx, shared.VISITOR_AREA, shared.UNKNOWN, 1); err != nil {
				zlog.Error(err.Error())
				return model.ResultFail()
			}
		}
		if _, err := shared.IncrByCtx(ctx, shared.BLOG_VIEWS_COUNT, 1); err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		if _, err := shared.SAddCtx(ctx, shared.UNIQUE_VISITOR, md5); err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
	}
	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) GetBlogHomeInfo(ctx context.Context) model.ResultVO {
	var articleCount, categoryCount, tagCount, talkCount int64
	engine := ormInit.GetEngine().Context(ctx)
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

	reData, err := shared.GetCtx(ctx, shared.BLOG_VIEWS_COUNT)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var viewCount int
	if reData != "" {
		viewCount, _ = strconv.Atoi(reData)
	} else {
		viewCount = 0
	}
	websiteConfigResult := b.GetWebsiteConfig(ctx)
	websiteConfig, ok := websiteConfigResult.Data.(model.WebsiteConfigDTO)
	if !ok {
		return websiteConfigResult
	}
	return model.ResultOkWithData(model.BenetnaschHomeInfoDTO{
		ArticleCount:    articleCount,
		CategoryCount:   categoryCount,
		TagCount:        tagCount,
		TalkCount:       talkCount,
		ViewCount:       viewCount,
		WebsiteConfigDT: websiteConfig,
	})
}

func (b *MyBenetnaschInfoService) GetWebsiteConfig(ctx context.Context) model.ResultVO {
	var webConfig model.WebsiteConfigDTO
	var config string
	websiteConfig, err := shared.GetCtx(ctx, shared.WEBSITE_CONFIG)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if websiteConfig != "" {
		if err := json.Unmarshal([]byte(websiteConfig), &webConfig); err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
	} else {
		_, err := ormInit.GetEngine().Context(ctx).SQL("select config from t_website_config where id = 1").Get(&config)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}

		if err := json.Unmarshal([]byte(config), &webConfig); err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}

		if err := shared.SetCtx(ctx, shared.WEBSITE_CONFIG, config); err != nil {
			zlog.Error(err.Error())
		}
	}
	return model.ResultOkWithData(webConfig)
}

func (b *MyBenetnaschInfoService) GetBlogBackInfo(ctx context.Context) model.ResultVO {
	viewCount, err := shared.GetCtx(ctx, shared.BLOG_VIEWS_COUNT)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	count, err := strconv.Atoi(viewCount)
	if err != nil {
		zlog.Error(err.Error())
	}
	engine := ormInit.GetEngine().Context(ctx)
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
	uniqueViews := listUniqueViews(ctx)
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
	articleMap, err := shared.ZReverseRangeWithScoreCtx(ctx, shared.ARTICLE_VIEWS_COUNT, 0, 4)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
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
		articleRankDTOs := b.listArticleRank(ctx, articleMap)
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

	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Exec("update t_website_config set config = ? where id = 1", string(m))
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage(err.Error())
	}
	if err := shared.SetCtx(c.Request.Context(), shared.WEBSITE_CONFIG, string(m)); err != nil {
		zlog.Error(err.Error())
	}

	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) GetAbout(ctx context.Context) model.ResultVO {
	var aboutDTO model.AboutDTO
	about, err := shared.GetCtx(ctx, shared.ABOUT)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if about != "" {
		err := json.Unmarshal([]byte(about), &aboutDTO)
		if err != nil {
			zlog.Error(err.Error())
		}
	} else {
		var abt entity.TAbout
		_, err := ormInit.GetEngine().Context(ctx).ID(shared.DEFAULT_ABOUT_ID).Get(&abt)
		if err != nil {
			zlog.Error(err.Error())
		}

		err = json.Unmarshal([]byte(abt.Content), &aboutDTO)
		if err != nil {
			zlog.Error(err.Error())
		}

		if err := shared.SetCtx(ctx, shared.ABOUT, abt.Content); err != nil {
			zlog.Error(err.Error())
		}
	}
	return model.ResultOkWithData(aboutDTO)
}

func (b *MyBenetnaschInfoService) UpdateAbout(c *gin.Context) model.ResultVO {
	var abt model.AboutVO
	if err := c.ShouldBind(&abt); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}

	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Exec("update t_about set config = ? where id = 1", abt.Content)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFailWithMessage(err.Error())
	}
	if err := shared.SetCtx(c.Request.Context(), shared.ABOUT, abt.Content); err != nil {
		zlog.Error(err.Error())
	}

	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) SaveBlogPhotoAlbumCover(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) listArticleRank(ctx context.Context, hm map[interface{}]float64) []model.ArticleRankDTO {
	var articleIds []int
	for k := range hm {
		id, err := strconv.Atoi(k.(string))
		if err != nil {
			zlog.Error(err.Error())
		}
		articleIds = append(articleIds, id)
	}
	var articles []entity.TArticle
	err := ormInit.GetEngine().Context(ctx).Select("id, article_title").In("id", articleIds).Find(&articles)
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
