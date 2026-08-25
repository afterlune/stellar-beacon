package service

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/shared"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
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
	listArticleRank(ctx context.Context, hm map[interface{}]float64) ([]model.ArticleRankDTO, error)
}

type MyBenetnaschInfoService struct {
	site       port.SiteInfoRepository
	articles   port.ArticleRepository
	categories port.CategoryRepository
	tags       port.TagRepository
}

func NewBenetnaschInfoService(site port.SiteInfoRepository, articles port.ArticleRepository, categories port.CategoryRepository, tags port.TagRepository) *MyBenetnaschInfoService {
	return &MyBenetnaschInfoService{site: site, articles: articles, categories: categories, tags: tags}
}

func (b *MyBenetnaschInfoService) siteRepository() port.SiteInfoRepository {
	if b.site != nil {
		return b.site
	}
	return siteInfoRepo
}

func (b *MyBenetnaschInfoService) articleRepository() port.ArticleRepository {
	if b.articles != nil {
		return b.articles
	}
	return articleRepo
}

func (b *MyBenetnaschInfoService) categoryRepository() port.CategoryRepository {
	if b.categories != nil {
		return b.categories
	}
	return categoryRepo
}

func (b *MyBenetnaschInfoService) tagRepository() port.TagRepository {
	if b.tags != nil {
		return b.tags
	}
	return tagRepo
}

func (b *MyBenetnaschInfoService) GetBenetnaschHomeInfo() model.ResultVO {
	return model.ResultVO{}
}

func (b *MyBenetnaschInfoService) Report(req *http.Request) model.ResultVO {
	ctx := req.Context()
	md5 := shared.GetMD5(shared.GetRedisId(req))
	seen, err := shared.SIsMemberCtx(ctx, shared.UNIQUE_VISITOR, md5)
	if err != nil {
		slog.Error("record unique visitor failed", "error", err)
		return model.ResultFail()
	}
	if !seen {
		ipSource := shared.GetIpSource(shared.GetIpAddress(req))
		if ipSource == "" {
			ipSource = shared.UNKNOWN
		}
		if _, err := shared.HIncrByCtx(ctx, shared.VISITOR_AREA, ipSource, 1); err != nil {
			slog.Error("increment visitor area failed", "error", err)
			return model.ResultFail()
		}
		if _, err := shared.IncrByCtx(ctx, shared.BLOG_VIEWS_COUNT, 1); err != nil {
			slog.Error("increment blog view count failed", "error", err)
			return model.ResultFail()
		}
		if _, err := shared.SAddCtx(ctx, shared.UNIQUE_VISITOR, md5); err != nil {
			slog.Error("record unique visitor failed", "error", err)
			return model.ResultFail()
		}
	}
	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) GetBlogHomeInfo(ctx context.Context) model.ResultVO {
	articles, err := b.siteRepository().CountArticles(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	categories, err := b.siteRepository().CountCategories(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	tags, err := b.siteRepository().CountTags(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	talks, err := b.siteRepository().CountTalks(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	viewCount, err := shared.GetCtx(ctx, shared.BLOG_VIEWS_COUNT)
	if err != nil {
		return model.ResultFail()
	}
	views, _ := strconv.Atoi(viewCount)
	config := b.GetWebsiteConfig(ctx)
	if !config.Flag {
		return config
	}
	websiteConfig, ok := config.Data.(model.WebsiteConfigDTO)
	if !ok {
		return model.ResultFail()
	}
	return model.ResultOkWithData(model.BenetnaschHomeInfoDTO{
		ArticleCount: articles, CategoryCount: categories, TagCount: tags,
		TalkCount: talks, ViewCount: views, WebsiteConfigDT: websiteConfig,
	})
}

func (b *MyBenetnaschInfoService) GetWebsiteConfig(ctx context.Context) model.ResultVO {
	var configDTO model.WebsiteConfigDTO
	cached, err := shared.GetCtx(ctx, shared.WEBSITE_CONFIG)
	if err != nil {
		return model.ResultFail()
	}
	config := cached
	if config == "" {
		config, err = b.siteRepository().GetWebsiteConfig(ctx)
		if err != nil {
			return model.ResultFromError(err)
		}
		if err := shared.SetCtx(ctx, shared.WEBSITE_CONFIG, config); err != nil {
			slog.Error("cache website configuration failed", "error", err)
		}
	}
	if err := json.Unmarshal([]byte(config), &configDTO); err != nil {
		slog.Error("decode website configuration failed", "error", err)
		return model.ResultFail()
	}
	return model.ResultOkWithData(configDTO)
}

func (b *MyBenetnaschInfoService) GetBlogBackInfo(ctx context.Context) model.ResultVO {
	viewCount, err := shared.GetCtx(ctx, shared.BLOG_VIEWS_COUNT)
	if err != nil {
		return model.ResultFail()
	}
	views, _ := strconv.Atoi(viewCount)
	messageCount, err := b.siteRepository().CountComments(ctx, 2)
	if err != nil {
		return model.ResultFromError(err)
	}
	userCount, err := b.siteRepository().CountUsers(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	articleCount, err := b.siteRepository().CountArticles(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	uniqueViews, err := b.listUniqueViews(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	articleStatistics, err := b.articleRepository().ListArticleStatistics(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	categories, err := b.categoryRepository().List(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	tags, err := b.tagRepository().List(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	tagDTOs := make([]model.TagDTO, 0, len(tags))
	for _, tag := range tags {
		if tag != nil {
			tagDTOs = append(tagDTOs, *tag)
		}
	}
	articleMap, err := shared.ZReverseRangeWithScoreCtx(ctx, shared.ARTICLE_VIEWS_COUNT, 0, 4)
	if err != nil {
		return model.ResultFail()
	}
	data := model.BenetnaschBackInfoDTO{
		ArticleStatisticsDTOs: articleStatistics,
		TagDTOs:               tagDTOs,
		ViewsCount:            views,
		MessageCount:          int(messageCount),
		UserCount:             int(userCount),
		ArticleCount:          int(articleCount),
		CategoryDTOs:          categories,
		UniqueViewDTOs:        uniqueViews,
	}
	if len(articleMap) > 0 {
		data.ArticleRankDTOs, err = b.listArticleRank(ctx, articleMap)
		if err != nil {
			return model.ResultFromError(err)
		}
	}
	return model.ResultOkWithData(data)
}

func (b *MyBenetnaschInfoService) UpdateWebsiteConfig(c *gin.Context) model.ResultVO {
	var vo model.WebsiteConfigVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	data, err := json.Marshal(&vo)
	if err != nil {
		return model.ResultFail()
	}
	if err := b.siteRepository().UpdateWebsiteConfig(c.Request.Context(), string(data)); err != nil {
		return model.ResultFromError(err)
	}
	if err := shared.SetCtx(c.Request.Context(), shared.WEBSITE_CONFIG, string(data)); err != nil {
		slog.Error("cache website configuration failed", "error", err)
	}
	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) GetAbout(ctx context.Context) model.ResultVO {
	var about model.AboutDTO
	cached, err := shared.GetCtx(ctx, shared.ABOUT)
	if err != nil {
		return model.ResultFail()
	}
	content := cached
	if content == "" {
		content, err = b.siteRepository().GetAbout(ctx, shared.DEFAULT_ABOUT_ID)
		if err != nil {
			return model.ResultFromError(err)
		}
		if err := shared.SetCtx(ctx, shared.ABOUT, content); err != nil {
			slog.Error("cache about content failed", "error", err)
		}
	}
	if err := json.Unmarshal([]byte(content), &about); err != nil {
		return model.ResultFail()
	}
	return model.ResultOkWithData(about)
}

func (b *MyBenetnaschInfoService) UpdateAbout(c *gin.Context) model.ResultVO {
	var vo model.AboutVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := b.siteRepository().UpdateAbout(c.Request.Context(), shared.DEFAULT_ABOUT_ID, vo.Content); err != nil {
		return model.ResultFromError(err)
	}
	if err := shared.SetCtx(c.Request.Context(), shared.ABOUT, vo.Content); err != nil {
		slog.Error("cache about content failed", "error", err)
	}
	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) SaveBlogPhotoAlbumCover(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func (b *MyBenetnaschInfoService) listUniqueViews(ctx context.Context) ([]model.UniqueViewDTO, error) {
	start := timeNow().Add(-7 * 24 * time.Hour).Format("2006-01-02")
	end := timeNow().Format("2006-01-02")
	views, err := b.siteRepository().ListUniqueViews(ctx, start, end)
	if err != nil {
		return nil, err
	}
	result := make([]model.UniqueViewDTO, 0, len(views))
	for _, view := range views {
		result = append(result, model.UniqueViewDTO{Day: view.Day, ViewsCount: view.ViewsCount})
	}
	return result, nil
}

func (b *MyBenetnaschInfoService) listArticleRank(ctx context.Context, hm map[interface{}]float64) ([]model.ArticleRankDTO, error) {
	ids := make([]int, 0, len(hm))
	for key := range hm {
		id, err := strconv.Atoi(fmt.Sprint(key))
		if err == nil {
			ids = append(ids, id)
		}
	}
	articles, err := b.siteRepository().ListArticleRank(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]model.ArticleRankDTO, 0, len(articles))
	for _, article := range articles {
		result = append(result, model.ArticleRankDTO{ArticleTitle: article.ArticleTitle, ViewsCount: int(hm[strconv.Itoa(article.Id)])})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ViewsCount > result[j].ViewsCount })
	return result, nil
}

var timeNow = time.Now
