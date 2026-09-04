package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

type BenetnaschInfoService interface {
	GetBenetnaschHomeInfo() port.ResultVO
	Report(req *http.Request) port.ResultVO
	GetBlogHomeInfo(ctx context.Context) port.ResultVO
	GetWebsiteConfig(ctx context.Context) port.ResultVO
	GetBlogBackInfo(ctx context.Context) port.ResultVO
	UpdateWebsiteConfig(c port.Request) port.ResultVO
	GetAbout(ctx context.Context) port.ResultVO
	UpdateAbout(c port.Request) port.ResultVO
	SaveBlogPhotoAlbumCover(c port.Request) port.ResultVO
	listArticleRank(ctx context.Context, hm map[string]float64) ([]port.ArticleRankDTO, error)
}

type MyBenetnaschInfoService struct {
	site       port.SiteInfoRepository
	articles   port.ArticleRepository
	categories port.CategoryRepository
	tags       port.TagRepository
	cache      port.Cache
	visitor    port.VisitorResolver
}

func NewBenetnaschInfoService(deps BenetnaschInfoServiceDeps) (*MyBenetnaschInfoService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyBenetnaschInfoService{
		site:       deps.Site,
		articles:   deps.Articles,
		categories: deps.Categories,
		tags:       deps.Tags,
		cache:      deps.Cache,
		visitor:    deps.Visitor,
	}, nil
}

func (b *MyBenetnaschInfoService) siteRepository() port.SiteInfoRepository {
	return b.site
}

func (b *MyBenetnaschInfoService) articleRepository() port.ArticleRepository {
	return b.articles
}

func (b *MyBenetnaschInfoService) categoryRepository() port.CategoryRepository {
	return b.categories
}

func (b *MyBenetnaschInfoService) tagRepository() port.TagRepository {
	return b.tags
}

func (b *MyBenetnaschInfoService) GetBenetnaschHomeInfo() port.ResultVO {
	return port.ResultVO{}
}

func (b *MyBenetnaschInfoService) Report(req *http.Request) port.ResultVO {
	if b.cache == nil || b.visitor == nil {
		return port.ResultFail()
	}
	ctx := req.Context()
	identity, err := b.visitor.Resolve(ctx, req)
	if err != nil {
		return port.ResultFail()
	}
	fingerprint := identity.Fingerprint
	seen, err := b.cache.SIsMember(ctx, support.UniqueVisitor, fingerprint)
	if err != nil {
		slog.Error("record unique visitor failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	if !seen {
		ipSource := identity.Region
		if ipSource == "" {
			ipSource = support.Unknown
		}
		if _, err := b.cache.HIncrBy(ctx, support.VisitorArea, ipSource, 1); err != nil {
			slog.Error("increment visitor area failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
		if _, err := b.cache.IncrBy(ctx, support.BlogViewsCount, 1); err != nil {
			slog.Error("increment blog view count failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
		if _, err := b.cache.SAdd(ctx, support.UniqueVisitor, fingerprint); err != nil {
			slog.Error("record unique visitor failed", "error_code", apperrors.SafeCode(err))
			return port.ResultFail()
		}
	}
	return port.ResultOk()
}

func (b *MyBenetnaschInfoService) GetBlogHomeInfo(ctx context.Context) port.ResultVO {
	articles, err := b.siteRepository().CountArticles(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	categories, err := b.siteRepository().CountCategories(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	tags, err := b.siteRepository().CountTags(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	talks, err := b.siteRepository().CountTalks(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	viewCount := "0"
	if b.cache != nil {
		viewCount, err = b.cache.Get(ctx, support.BlogViewsCount)
		if errors.Is(err, port.ErrCacheMiss) {
			viewCount = "0"
		} else if err != nil {
			slog.WarnContext(ctx, "load blog view count failed", "error_code", apperrors.SafeCode(err))
			viewCount = "0"
		}
	}
	views := parseCachedCount(ctx, "blog_views", viewCount)
	config := b.GetWebsiteConfig(ctx)
	if !config.Flag {
		return config
	}
	websiteConfig, ok := config.Data.(port.WebsiteConfigDTO)
	if !ok {
		return port.ResultFail()
	}
	return port.ResultOkWithData(port.BenetnaschHomeInfoDTO{
		ArticleCount: articles, CategoryCount: categories, TagCount: tags,
		TalkCount: talks, ViewCount: views, WebsiteConfigDT: websiteConfig,
	})
}

func (b *MyBenetnaschInfoService) GetWebsiteConfig(ctx context.Context) port.ResultVO {
	var configDTO port.WebsiteConfigDTO
	var err error
	config := ""
	if b.cache != nil {
		config, err = b.cache.Get(ctx, support.WebsiteConfig)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(ctx, "read website configuration cache failed", "error_code", apperrors.SafeCode(err))
			config = ""
		}
	}
	if config == "" {
		config, err = b.siteRepository().GetWebsiteConfig(ctx)
		if err != nil {
			return port.ResultFromError(err)
		}
		if b.cache != nil {
			if err := b.cache.Set(ctx, support.WebsiteConfig, config, 0); err != nil {
				slog.Error("cache website configuration failed", "error_code", apperrors.SafeCode(err))
			}
		}
	}
	if err := json.Unmarshal([]byte(config), &configDTO); err != nil {
		slog.Error("decode website configuration failed", "error_code", apperrors.SafeCode(err))
		return port.ResultFail()
	}
	return port.ResultOkWithData(configDTO)
}

func (b *MyBenetnaschInfoService) GetBlogBackInfo(ctx context.Context) port.ResultVO {
	var err error
	viewCount := "0"
	if b.cache != nil {
		viewCount, err = b.cache.Get(ctx, support.BlogViewsCount)
		if errors.Is(err, port.ErrCacheMiss) {
			viewCount = "0"
		} else if err != nil {
			slog.WarnContext(ctx, "load blog view count failed", "error_code", apperrors.SafeCode(err))
			viewCount = "0"
		}
	}
	views := parseCachedCount(ctx, "blog_views", viewCount)
	messageCount, err := b.siteRepository().CountComments(ctx, 2)
	if err != nil {
		return port.ResultFromError(err)
	}
	userCount, err := b.siteRepository().CountUsers(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	articleCount, err := b.siteRepository().CountArticles(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	uniqueViews, err := b.listUniqueViews(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	articleStatistics, err := b.articleRepository().ListArticleStatistics(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	categories, err := b.categoryRepository().List(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	tags, err := b.tagRepository().List(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	tagDTOs := make([]port.TagDTO, 0, len(tags))
	for _, tag := range tags {
		if tag != nil {
			tagDTOs = append(tagDTOs, *tag)
		}
	}
	articleMap := map[string]float64{}
	if b.cache != nil {
		articleMap, err = b.cache.ZRevRangeWithScores(ctx, support.ArticleViewsCount, 0, 4)
		if err != nil {
			slog.WarnContext(ctx, "load article view ranking failed", "error_code", apperrors.SafeCode(err))
			articleMap = map[string]float64{}
		}
	}
	data := port.BenetnaschBackInfoDTO{
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
			return port.ResultFromError(err)
		}
	}
	return port.ResultOkWithData(data)
}

func (b *MyBenetnaschInfoService) UpdateWebsiteConfig(c port.Request) port.ResultVO {
	var vo port.WebsiteConfigVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	data, err := json.Marshal(&vo)
	if err != nil {
		return port.ResultFail()
	}
	if err := b.siteRepository().UpdateWebsiteConfig(c.Context(), string(data)); err != nil {
		return port.ResultFromError(err)
	}
	if b.cache != nil {
		if err := b.cache.Set(c.Context(), support.WebsiteConfig, string(data), 0); err != nil {
			slog.Error("cache website configuration failed", "error_code", apperrors.SafeCode(err))
		}
	}
	return port.ResultOk()
}

func (b *MyBenetnaschInfoService) GetAbout(ctx context.Context) port.ResultVO {
	var err error
	content := ""
	fromCache := false
	if b.cache != nil {
		content, err = b.cache.Get(ctx, support.About)
		if err != nil && !errors.Is(err, port.ErrCacheMiss) {
			slog.WarnContext(ctx, "read about cache failed", "error_code", apperrors.SafeCode(err))
			content = ""
		} else if err == nil && content != "" {
			fromCache = true
		}
	}
	if content == "" {
		content, err = b.siteRepository().GetAbout(ctx, support.DefaultAboutID)
		if err != nil {
			return port.ResultFromError(err)
		}
	}
	about, normalized, err := decodeAboutContent(content)
	if err != nil {
		return port.ResultFail()
	}
	if b.cache != nil && (!fromCache || normalized != content) {
		if err := b.cache.Set(ctx, support.About, normalized, 0); err != nil {
			slog.Error("cache about content failed", "error_code", apperrors.SafeCode(err))
		}
	}
	return port.ResultOkWithData(about)
}

func (b *MyBenetnaschInfoService) UpdateAbout(c port.Request) port.ResultVO {
	var vo port.AboutVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	data, err := json.Marshal(port.AboutDTO{Content: vo.Content})
	if err != nil {
		return port.ResultFail()
	}
	serialized := string(data)
	if err := b.siteRepository().UpdateAbout(c.Context(), support.DefaultAboutID, serialized); err != nil {
		return port.ResultFromError(err)
	}
	if b.cache != nil {
		if err := b.cache.Set(c.Context(), support.About, serialized, 0); err != nil {
			slog.Error("cache about content failed", "error_code", apperrors.SafeCode(err))
		}
	}
	return port.ResultOk()
}

func (b *MyBenetnaschInfoService) SaveBlogPhotoAlbumCover(c port.Request) port.ResultVO {
	return port.ResultOk()
}

func (b *MyBenetnaschInfoService) listUniqueViews(ctx context.Context) ([]port.UniqueViewDTO, error) {
	start := timeNow().Add(-7 * 24 * time.Hour).Format("2006-01-02")
	end := timeNow().Format("2006-01-02")
	views, err := b.siteRepository().ListUniqueViews(ctx, start, end)
	if err != nil {
		return nil, err
	}
	result := make([]port.UniqueViewDTO, 0, len(views))
	for _, view := range views {
		result = append(result, port.UniqueViewDTO{Day: view.Day, ViewsCount: view.ViewsCount})
	}
	return result, nil
}

func (b *MyBenetnaschInfoService) listArticleRank(ctx context.Context, hm map[string]float64) ([]port.ArticleRankDTO, error) {
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
	result := make([]port.ArticleRankDTO, 0, len(articles))
	for _, article := range articles {
		result = append(result, port.ArticleRankDTO{ArticleTitle: article.ArticleTitle, ViewsCount: int(hm[strconv.Itoa(article.Id)])})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ViewsCount > result[j].ViewsCount })
	return result, nil
}

var timeNow = time.Now

func decodeAboutContent(serialized string) (port.AboutDTO, string, error) {
	var about port.AboutDTO
	if err := json.Unmarshal([]byte(serialized), &about); err == nil {
		return about, serialized, nil
	} else if looksLikeJSON(serialized) {
		return port.AboutDTO{}, "", err
	}

	// Releases before the JSON envelope fix stored Markdown directly. Read it
	// once for compatibility and normalize only the cache; the database is not
	// implicitly migrated by a read request.
	data, err := json.Marshal(port.AboutDTO{Content: serialized})
	if err != nil {
		return port.AboutDTO{}, "", err
	}
	return port.AboutDTO{Content: serialized}, string(data), nil
}

func looksLikeJSON(value string) bool {
	trimmed := strings.TrimSpace(value)
	// About is stored as a JSON object. Markdown commonly starts with '['
	// (links and lists), so only an object-shaped value is treated as malformed
	// structured data instead of legacy plain text.
	return strings.HasPrefix(trimmed, "{")
}

func parseCachedCount(ctx context.Context, metric, value string) int {
	count, err := strconv.Atoi(value)
	if err != nil {
		slog.WarnContext(ctx, "invalid cached metric count", "metric", metric, "reason", "not_an_integer")
		return 0
	}
	if count < 0 {
		slog.WarnContext(ctx, "cached metric count is negative", "metric", metric)
		return 0
	}
	return count
}
