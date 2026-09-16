package service

import (
	"context"
	"errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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
	SaveBlogPhotoAlbumCover(c *gin.Context) model.ResultVO
	listArticleRank(ctx context.Context, hm map[string]float64) ([]model.ArticleRankDTO, error)
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

func (b *MyStellarBeaconInfoService) GetDashboardAnalytics(ctx context.Context, rangeValue, areaType string) model.ResultVO {
	if rangeValue != "30d" && rangeValue != "12m" {
		rangeValue = "7d"
	}
	now := timeNow()
	unit := "day"
	start := now.AddDate(0, 0, -6)
	switch rangeValue {
	case "30d":
		start = now.AddDate(0, 0, -29)
	case "12m":
		unit = "month"
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -11, 0)
	}
	end := now.AddDate(0, 0, 1)
	historic, err := b.siteRepository().ListUniqueViews(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return model.ResultFromError(err)
	}
	historicByDay := make(map[string]int, len(historic))
	for _, item := range historic {
		historicByDay[item.Day] = item.ViewsCount
	}

	trend := make([]model.DashboardTrendDTO, 0)
	todayViews := 0
	monthViews := 0
	if unit == "day" {
		for cursor := start; !cursor.After(now); cursor = cursor.AddDate(0, 0, 1) {
			period := cursor.Format("2006-01-02")
			views := b.dailyViews(ctx, period, historicByDay[period])
			trend = append(trend, model.DashboardTrendDTO{Period: period, Views: views})
			if period == now.Format("2006-01-02") {
				todayViews = views
			}
		}
	} else {
		for cursor := start; !cursor.After(now); cursor = cursor.AddDate(0, 1, 0) {
			period := cursor.Format("2006-01")
			views := 0
			for day := cursor; day.Before(cursor.AddDate(0, 1, 0)) && !day.After(now); day = day.AddDate(0, 0, 1) {
				key := day.Format("2006-01-02")
				views += b.dailyViews(ctx, key, historicByDay[key])
			}
			trend = append(trend, model.DashboardTrendDTO{Period: period, Views: views})
		}
		todayViews = b.dailyViews(ctx, now.Format("2006-01-02"), historicByDay[now.Format("2006-01-02")])
	}
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for day := monthStart; !day.After(now); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		monthViews += b.dailyViews(ctx, key, historicByDay[key])
	}

	viewCount := 0
	if raw, cacheErr := b.cache.Get(ctx, BlogViewsCount); cacheErr == nil {
		viewCount, _ = strconv.Atoi(raw)
	}
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
	categories, err := b.categoryRepository().List(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	tags, err := b.tagRepository().List(ctx)
	if err != nil {
		return model.ResultFromError(err)
	}
	categoryStats := make([]model.DashboardDistributionDTO, 0, len(categories))
	for _, category := range categories {
		categoryStats = append(categoryStats, model.DashboardDistributionDTO{Name: category.CategoryName, Value: int64(category.ArticleCount)})
	}
	tagStats := make([]model.DashboardDistributionDTO, 0, len(tags))
	for _, tag := range tags {
		if tag != nil {
			tagStats = append(tagStats, model.DashboardDistributionDTO{Name: tag.TagName, Value: int64(tag.Count)})
		}
	}
	regions := b.dashboardRegions(ctx, areaType)
	articleRank := make([]model.DashboardArticleRankDTO, 0)
	if b.cache != nil {
		articleMap, cacheErr := b.cache.ZRevRangeWithScores(ctx, ArticleViewsCount, 0, 7)
		if cacheErr == nil && len(articleMap) > 0 {
			ids := make([]int, 0, len(articleMap))
			for key := range articleMap {
				if id, parseErr := strconv.Atoi(key); parseErr == nil {
					ids = append(ids, id)
				}
			}
			articles, listErr := b.siteRepository().ListArticleRank(ctx, ids)
			if listErr != nil {
				return model.ResultFromError(listErr)
			}
			for _, article := range articles {
				articleRank = append(articleRank, model.DashboardArticleRankDTO{Id: article.Id, Title: article.ArticleTitle, Views: int(articleMap[strconv.Itoa(article.Id)])})
			}
			sort.Slice(articleRank, func(i, j int) bool { return articleRank[i].Views > articleRank[j].Views })
		}
	}
	growth := model.DashboardGrowthDTO{Trend: make([]model.DashboardGrowthTrendDTO, 0)}
	if b.newsletter != nil && b.growth != nil {
		newsletterStats, statsErr := b.newsletter.Stats(ctx)
		if statsErr != nil {
			return model.ResultFromError(statsErr)
		}
		growthRows, growthErr := b.growth.SummaryByPeriod(ctx, start, unit)
		if growthErr != nil {
			return model.ResultFromError(growthErr)
		}
		deliveryRows, deliveryErr := b.newsletter.DeliveryTrend(ctx, start, unit)
		if deliveryErr != nil {
			return model.ResultFromError(deliveryErr)
		}
		growth = dashboardGrowthDTO(newsletterStats, growthRows, deliveryRows, start, now, unit)
	}
	return model.ResultOkWithData(model.DashboardAnalyticsDTO{
		Range: rangeValue,
		Unit:  unit,
		Overview: model.DashboardOverviewDTO{
			TotalViews:   viewCount,
			TodayViews:   todayViews,
			MonthViews:   monthViews,
			UserCount:    int(userCount),
			ArticleCount: int(articleCount),
			MessageCount: int(messageCount),
		},
		Trend:       trend,
		Regions:     regions,
		Categories:  categoryStats,
		Tags:        tagStats,
		ArticleRank: articleRank,
		Growth:      growth,
		GeneratedAt: now,
	})
}

func (b *MyStellarBeaconInfoService) dailyViews(ctx context.Context, day string, fallback int) int {
	if raw, err := b.cache.Get(ctx, DailyViewsPrefix+day); err == nil {
		value, parseErr := strconv.Atoi(raw)
		if parseErr == nil {
			return value
		}
	}
	return fallback
}

func (b *MyStellarBeaconInfoService) dashboardRegions(ctx context.Context, areaType string) []model.DashboardRegionDTO {
	if areaType != "visitors" {
		areaType = "users"
	}
	regions := make([]model.DashboardRegionDTO, 0)
	if areaType == "users" {
		raw, err := b.cache.Get(ctx, UserArea)
		if err != nil || raw == "" {
			return regions
		}
		var values []model.UserAreaDTO
		if json.Unmarshal([]byte(raw), &values) != nil {
			return regions
		}
		for _, value := range values {
			regions = append(regions, regionDTO(value.Name, value.Name, value.Value))
		}
		return regions
	}
	values, err := b.cache.HGetAll(ctx, VisitorArea)
	if err != nil {
		return regions
	}
	for rawName, rawValue := range values {
		count, parseErr := strconv.ParseInt(rawValue, 10, 64)
		if parseErr != nil {
			continue
		}
		label := rawName
		parts := strings.Split(rawName, "|")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			label = strings.TrimSpace(parts[0])
		}
		regions = append(regions, regionDTO(label, rawName, count))
	}
	return regions
}

func regionDTO(name, label string, value int64) model.DashboardRegionDTO {
	code := ""
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "中国", "china", "cn":
		code = "CN"
	case "美国", "united states", "us":
		code = "US"
	case "日本", "japan", "jp":
		code = "JP"
	case "英国", "united kingdom", "uk":
		code = "GB"
	case "加拿大", "canada", "ca":
		code = "CA"
	case "澳大利亚", "australia", "au":
		code = "AU"
	}
	return model.DashboardRegionDTO{Name: name, Label: label, Code: code, Value: value}
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

func (b *MyStellarBeaconInfoService) SaveBlogPhotoAlbumCover(c *gin.Context) model.ResultVO {
	return model.ResultOk()
}

func (b *MyStellarBeaconInfoService) listUniqueViews(ctx context.Context) ([]model.UniqueViewDTO, error) {
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

func (b *MyStellarBeaconInfoService) listArticleRank(ctx context.Context, hm map[string]float64) ([]model.ArticleRankDTO, error) {
	ids := make([]int, 0, len(hm))
	for key := range hm {
		id, err := strconv.Atoi(key)
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
