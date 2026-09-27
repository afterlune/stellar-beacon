package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

type StellarBeaconInfoService interface {
	Report(req *http.Request) model.ResultVO
	GetBlogHomeInfo(ctx context.Context) model.ResultVO
	GetWebsiteConfig(ctx context.Context) model.ResultVO
	GetBlogBackInfo(ctx context.Context) model.ResultVO
	GetDashboardAnalytics(ctx context.Context, rangeValue, areaType string) model.ResultVO
	UpdateWebsiteConfig(c *gin.Context) model.ResultVO
	GetAbout(ctx context.Context) model.ResultVO
	UpdateAbout(c *gin.Context) model.ResultVO
}

type MyStellarBeaconInfoService struct {
	site       port.SiteInfoRepository
	articles   port.ArticleRepository
	categories port.CategoryRepository
	tags       port.TagRepository
	cache      port.Cache
	visitor    port.VisitorResolver
	newsletter port.NewsletterRepository
	growth     port.GrowthRepository
}

func NewStellarBeaconInfoService(deps StellarBeaconInfoServiceDeps) (*MyStellarBeaconInfoService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyStellarBeaconInfoService{
		site:       deps.Site,
		articles:   deps.Articles,
		categories: deps.Categories,
		tags:       deps.Tags,
		cache:      deps.Cache,
		visitor:    deps.Visitor,
		newsletter: deps.Newsletter,
		growth:     deps.Growth,
	}, nil
}

func (b *MyStellarBeaconInfoService) siteRepository() port.SiteInfoRepository {
	return b.site
}

func (b *MyStellarBeaconInfoService) articleRepository() port.ArticleRepository {
	return b.articles
}

func (b *MyStellarBeaconInfoService) categoryRepository() port.CategoryRepository {
	return b.categories
}

func (b *MyStellarBeaconInfoService) tagRepository() port.TagRepository {
	return b.tags
}

func (b *MyStellarBeaconInfoService) Report(req *http.Request) model.ResultVO {
	if b.cache == nil || b.visitor == nil {
		return model.ResultFail()
	}
	ctx := req.Context()
	identity, err := b.visitor.Resolve(ctx, req)
	if err != nil {
		return model.ResultFail()
	}
	fingerprint := identity.Fingerprint
	today := timeNow().Format("2006-01-02")
	dailyVisitorKey := DailyVisitorPrefix + today
	dailySeen, err := b.cache.SIsMember(ctx, dailyVisitorKey, fingerprint)
	if err != nil {
		slog.Error("record unique visitor failed", "error", err)
		return model.ResultFail()
	}
	if !dailySeen {
		if _, err := b.cache.SAdd(ctx, dailyVisitorKey, fingerprint); err != nil {
			slog.Error("record daily unique visitor failed", "error", err)
			return model.ResultFail()
		}
		if _, err := b.cache.Expire(ctx, dailyVisitorKey, 400*24*time.Hour); err != nil {
			slog.WarnContext(ctx, "expire daily unique visitor set failed", "error", err)
		}
		if _, err := b.cache.IncrBy(ctx, DailyViewsPrefix+today, 1); err != nil {
			slog.Error("increment daily blog view count failed", "error", err)
			return model.ResultFail()
		}
	}
	seen, err := b.cache.SIsMember(ctx, UniqueVisitor, fingerprint)
	if err != nil {
		slog.Error("record unique visitor failed", "error", err)
		return model.ResultFail()
	}
	if !seen {
		ipSource := identity.Region
		if ipSource == "" {
			ipSource = Unknown
		}
		if _, err := b.cache.HIncrBy(ctx, VisitorArea, ipSource, 1); err != nil {
			slog.Error("increment visitor area failed", "error", err)
			return model.ResultFail()
		}
		if _, err := b.cache.IncrBy(ctx, BlogViewsCount, 1); err != nil {
			slog.Error("increment blog view count failed", "error", err)
			return model.ResultFail()
		}
		if _, err := b.cache.SAdd(ctx, UniqueVisitor, fingerprint); err != nil {
			slog.Error("record unique visitor failed", "error", err)
			return model.ResultFail()
		}
	}
	return model.ResultOk()
}

func (b *MyStellarBeaconInfoService) GetBlogHomeInfo(ctx context.Context) model.ResultVO {
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
	viewCount := "0"
	if b.cache != nil {
		viewCount, err = b.cache.Get(ctx, BlogViewsCount)
		if errors.Is(err, port.ErrCacheMiss) {
			viewCount = "0"
		} else if err != nil {
			slog.WarnContext(ctx, "load blog view count failed", "error", err)
			viewCount = "0"
		}
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
	return model.ResultOkWithData(model.StellarBeaconHomeInfoDTO{
		ArticleCount: articles, CategoryCount: categories, TagCount: tags,
		TalkCount: talks, ViewCount: views, WebsiteConfigDT: websiteConfig,
	})
}

func (b *MyStellarBeaconInfoService) GetWebsiteConfig(ctx context.Context) model.ResultVO {
	var configDTO model.WebsiteConfigDTO
	var err error
	config := ""
	if b.cache != nil {
		config, err = b.cache.Get(ctx, WebsiteConfig)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(ctx, "read website configuration cache failed", "error", err)
			config = ""
		}
	}
	if config == "" {
		config, err = b.siteRepository().GetWebsiteConfig(ctx)
		if err != nil {
			return model.ResultFromError(err)
		}
		if b.cache != nil {
			if err := b.cache.Set(ctx, WebsiteConfig, config, 0); err != nil {
				slog.Error("cache website configuration failed", "error", err)
			}
		}
	}
	if err := json.Unmarshal([]byte(config), &configDTO); err != nil {
		slog.Error("decode website configuration failed", "error", err)
		return model.ResultFail()
	}
	return model.ResultOkWithData(configDTO)
}

func (b *MyStellarBeaconInfoService) GetBlogBackInfo(ctx context.Context) model.ResultVO {
	var err error
	viewCount := "0"
	if b.cache != nil {
		viewCount, err = b.cache.Get(ctx, BlogViewsCount)
		if errors.Is(err, port.ErrCacheMiss) {
			viewCount = "0"
		} else if err != nil {
			slog.WarnContext(ctx, "load blog view count failed", "error", err)
			viewCount = "0"
		}
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
	articleMap := map[string]float64{}
	if b.cache != nil {
		articleMap, err = b.cache.ZRevRangeWithScores(ctx, ArticleViewsCount, 0, 4)
		if err != nil {
			slog.WarnContext(ctx, "load article view ranking failed", "error", err)
			articleMap = map[string]float64{}
		}
	}
	data := model.StellarBeaconBackInfoDTO{
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

func (b *MyStellarBeaconInfoService) UpdateWebsiteConfig(c *gin.Context) model.ResultVO {
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
	if b.cache != nil {
		if err := b.cache.Set(c.Request.Context(), WebsiteConfig, string(data), 0); err != nil {
			slog.Error("cache website configuration failed", "error", err)
		}
	}
	return model.ResultOk()
}

func (b *MyStellarBeaconInfoService) GetAbout(ctx context.Context) model.ResultVO {
	var about model.AboutDTO
	var err error
	content := ""
	if b.cache != nil {
		content, err = b.cache.Get(ctx, About)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(ctx, "read about cache failed", "error", err)
			content = ""
		}
	}
	if content == "" {
		content, err = b.siteRepository().GetAbout(ctx, DefaultAboutID)
		if err != nil {
			return model.ResultFromError(err)
		}
		if b.cache != nil {
			if err := b.cache.Set(ctx, About, content, 0); err != nil {
				slog.Error("cache about content failed", "error", err)
			}
		}
	}
	if err := json.Unmarshal([]byte(content), &about); err != nil {
		// Older deployments stored the editor value directly instead of the
		// JSON document used by the public DTO. Keep those records readable
		// while all new writes use the canonical document shape below.
		about.Content = content
	}
	return model.ResultOkWithData(about)
}

func (b *MyStellarBeaconInfoService) UpdateAbout(c *gin.Context) model.ResultVO {
	var vo model.AboutVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	document, err := json.Marshal(model.AboutDTO(vo))
	if err != nil {
		return model.ResultFailWithMessage("关于内容格式不正确")
	}
	serialized := string(document)
	if err := b.siteRepository().UpdateAbout(c.Request.Context(), DefaultAboutID, serialized); err != nil {
		return model.ResultFromError(err)
	}
	if b.cache != nil {
		if err := b.cache.Set(c.Request.Context(), About, serialized, 0); err != nil {
			slog.Error("cache about content failed", "error", err)
		}
	}
	return model.ResultOk()
}

var timeNow = time.Now
