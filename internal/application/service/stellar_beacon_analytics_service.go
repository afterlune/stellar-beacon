package service

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"

	"github.com/goccy/go-json"
)

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
		activation, activationErr := b.growth.StudioActivationFunnel(ctx, start)
		if activationErr != nil {
			return model.ResultFromError(activationErr)
		}
		growth = dashboardGrowthDTO(newsletterStats, growthRows, deliveryRows, activation, start, now, unit)
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
