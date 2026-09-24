package service

import (
	"container/list"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type PlatformService interface {
	Feed(c *gin.Context) model.ResultVO
	Authors(c *gin.Context) model.ResultVO
	Author(c *gin.Context) model.ResultVO
	AuthorArticles(c *gin.Context) model.ResultVO
	AuthorTalks(c *gin.Context) model.ResultVO
	AuthorSeries(c *gin.Context) model.ResultVO
	TopicArticles(c *gin.Context) model.ResultVO
	Topics(c *gin.Context) model.ResultVO
	Dashboard(c *gin.Context) model.ResultVO
	SyncActivation(c *gin.Context) model.ResultVO
	GetProfile(c *gin.Context) model.ResultVO
	UpdateProfile(c *gin.Context) model.ResultVO

	ListOwnedArticles(c *gin.Context) model.ResultVO
	GetOwnedArticle(c *gin.Context) model.ResultVO
	SaveOwnedArticle(c *gin.Context) model.ResultVO
	DeleteOwnedArticles(c *gin.Context) model.ResultVO

	ListOwnedTalks(c *gin.Context) model.ResultVO
	GetOwnedTalk(c *gin.Context) model.ResultVO
	SaveOwnedTalk(c *gin.Context) model.ResultVO
	DeleteOwnedTalks(c *gin.Context) model.ResultVO

	ListOwnedSeries(c *gin.Context) model.ResultVO
	GetOwnedSeries(c *gin.Context) model.ResultVO
	SaveOwnedSeries(c *gin.Context) model.ResultVO
	DeleteOwnedSeries(c *gin.Context) model.ResultVO
	BatchPreviewContent(c *gin.Context) model.ResultVO
	BatchUpdateContentStatus(c *gin.Context) model.ResultVO
	BatchDeleteContent(c *gin.Context) model.ResultVO
	RetryScheduledPublication(c *gin.Context) model.ResultVO

	ListOwnedCategories(c *gin.Context) model.ResultVO
	SaveOwnedCategory(c *gin.Context) model.ResultVO
	DeleteOwnedCategory(c *gin.Context) model.ResultVO
	ListOwnedTags(c *gin.Context) model.ResultVO
	SaveOwnedTag(c *gin.Context) model.ResultVO
	DeleteOwnedTag(c *gin.Context) model.ResultVO

	Moderate(c *gin.Context) model.ResultVO
	DistributeArticle(c *gin.Context) model.ResultVO
	Upload(c *gin.Context) model.ResultVO
}

// reactionCounter and commentCounter are the narrow read shapes the public
// discovery payloads need. They are optional dependencies: when a deployment
// does not wire them the feed still renders, only the counters stay at zero.
type reactionCounter interface {
	Counts(ctx context.Context, articleIDs []int) (map[int]port.ReactionCounts, error)
}

type commentCounter interface {
	ListCommentCountsByTypeAndTopicIDs(ctx context.Context, commentType int, topicIDs []int) ([]*port.CommentCount, error)
}

type PlatformServiceDeps struct {
	Repo       port.PlatformRepository
	Articles   port.ArticleRepository
	Newsletter articleDistributor
	Storage    port.ObjectStorage
	Cache      port.Cache
	Reactions  reactionCounter
	Comments   commentCounter
}

type articleDistributor interface {
	EnqueueArticle(context.Context, int) error
}

type MyPlatformService struct {
	repo       port.PlatformRepository
	articles   port.ArticleRepository
	newsletter articleDistributor
	storage    port.ObjectStorage
	cache      port.Cache
	reactions  reactionCounter
	comments   commentCounter
}

func NewPlatformService(deps PlatformServiceDeps) (*MyPlatformService, error) {
	if deps.Repo == nil || deps.Articles == nil || deps.Storage == nil {
		return nil, fmt.Errorf("platform service dependencies are incomplete")
	}
	return &MyPlatformService{
		repo: deps.Repo, articles: deps.Articles, newsletter: deps.Newsletter,
		storage: deps.Storage, cache: deps.Cache, reactions: deps.Reactions, comments: deps.Comments,
	}, nil
}

func (s *MyPlatformService) platformRepo() port.PlatformRepository { return s.repo }
func (s *MyPlatformService) Feed(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	sort, ok := discoveryFeedSort(c.Query("sort"))
	if !ok {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	kind := strings.TrimSpace(c.Query("type"))
	featuredOnly := c.Query("featured") == "1" || c.Query("featured") == "true"
	if kind == "talk" {
		talks, count, err := s.platformRepo().ListFeedTalks(c.Request.Context(), current, size)
		if err != nil {
			return model.ResultFromError(err)
		}
		if len(talks) == 0 {
			return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
		}
		s.attachTalkCommentCounts(c.Request.Context(), talks)
		return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: count})
	}
	var articles []*port.ArticleCard
	var count int
	switch sort {
	case port.FeedSortHot:
		articles, count, err = s.platformRepo().ListFeedArticlesHot(c.Request.Context(), current, size)
	case port.FeedSortFeatured:
		articles, count, err = s.platformRepo().ListFeedArticles(c.Request.Context(), current, size, true)
	default:
		articles, count, err = s.platformRepo().ListFeedArticles(c.Request.Context(), current, size, featuredOnly)
	}
	if err != nil {
		return model.ResultFromError(err)
	}
	if len(articles) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	s.attachArticleReactionCounts(c.Request.Context(), articles)
	return model.ResultOkWithData(model.PageResultDTO{Records: articles, Count: count})
}

// discoveryFeedSort validates the feed sort key. An empty value keeps the
// historical recency ordering so existing callers are unaffected.
func discoveryFeedSort(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", port.FeedSortLatest:
		return port.FeedSortLatest, true
	case port.FeedSortHot, port.FeedSortFeatured:
		return strings.ToLower(strings.TrimSpace(value)), true
	default:
		return "", false
	}
}

func (s *MyPlatformService) Authors(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	sort, ok := discoveryAuthorSort(c.Query("sort"))
	if !ok {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	authors, count, err := s.platformRepo().ListAuthors(c.Request.Context(), current, size, optionalUserID(c), sort)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: authors, Count: count})
}

// discoveryAuthorSort validates the author-board ordering key. The historical
// article-count order stays the default for existing callers.
func discoveryAuthorSort(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", port.AuthorSortArticles:
		return port.AuthorSortArticles, true
	case port.AuthorSortFollowers, port.AuthorSortActive:
		return strings.ToLower(strings.TrimSpace(value)), true
	default:
		return "", false
	}
}
func (s *MyPlatformService) Author(c *gin.Context) model.ResultVO {
	author, err := s.platformRepo().GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(author)
}

func (s *MyPlatformService) AuthorArticles(c *gin.Context) model.ResultVO {
	author, err := s.platformRepo().GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	sort, ok := discoveryFeedSort(c.Query("sort"))
	if !ok || sort == port.FeedSortFeatured {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var articles []*port.ArticleCard
	var count int
	if sort == port.FeedSortHot {
		articles, count, err = s.platformRepo().ListAuthorArticlesHot(c.Request.Context(), author.Id, current, size)
	} else {
		articles, count, err = s.platformRepo().ListAuthorArticles(c.Request.Context(), author.Id, current, size)
	}
	if err != nil {
		return model.ResultFromError(err)
	}
	s.attachArticleReactionCounts(c.Request.Context(), articles)
	return model.ResultOkWithData(model.PageResultDTO{Records: articles, Count: count})
}

func (s *MyPlatformService) AuthorTalks(c *gin.Context) model.ResultVO {
	author, err := s.platformRepo().GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	talks, count, err := s.platformRepo().ListAuthorTalks(c.Request.Context(), author.Id, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: count})
}

func (s *MyPlatformService) AuthorSeries(c *gin.Context) model.ResultVO {
	author, err := s.platformRepo().GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	series, count, err := s.platformRepo().ListAuthorSeries(c.Request.Context(), author.Id, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: series, Count: count})
}

func (s *MyPlatformService) TopicArticles(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	articles, count, err := s.platformRepo().ListTopicArticles(c.Request.Context(), c.Param("topic"), c.Param("slug"), current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	s.attachArticleReactionCounts(c.Request.Context(), articles)
	return model.ResultOkWithData(model.PageResultDTO{Records: articles, Count: count})
}

// Topic overview sizes are clamped instead of rejected: the plaza is a
// presentational surface, so an odd size should degrade to a usable page.
const (
	topicOverviewSizeDefault = 12
	topicOverviewSizeMax     = 30
)

func (s *MyPlatformService) Topics(c *gin.Context) model.ResultVO {
	size := topicOverviewSizeDefault
	if raw := strings.TrimSpace(c.Query("size")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			size = parsed
		}
	}
	if size > topicOverviewSizeMax {
		size = topicOverviewSizeMax
	}
	overview, err := s.platformRepo().ListTopicOverview(c.Request.Context(), size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(overview)
}

// attachArticleReactionCounts decorates discovery cards with like/favourite
// totals. Counts are presentational: a ledger failure is logged and leaves the
// zero value instead of failing the whole read.
func (s *MyPlatformService) attachArticleReactionCounts(ctx context.Context, cards []*port.ArticleCard) {
	if s.reactions == nil || len(cards) == 0 {
		return
	}
	ids := make([]int, 0, len(cards))
	for _, card := range cards {
		if card != nil && card.Id > 0 {
			ids = append(ids, card.Id)
		}
	}
	counts, err := s.reactions.Counts(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "load discovery reaction counts failed", "error", err)
		return
	}
	for _, card := range cards {
		if card == nil {
			continue
		}
		totals := counts[card.Id]
		card.LikeCount = totals.LikeCount
		card.FavoriteCount = totals.FavoriteCount
	}
}

// attachTalkCommentCounts decorates feed talks with their approved-comment
// totals using the same aggregate the talk service publishes.
func (s *MyPlatformService) attachTalkCommentCounts(ctx context.Context, talks []*port.Talk) {
	if s.comments == nil || len(talks) == 0 {
		return
	}
	ids := make([]int, 0, len(talks))
	for _, talk := range talks {
		if talk != nil && talk.Id > 0 {
			ids = append(ids, talk.Id)
		}
	}
	counts, err := s.comments.ListCommentCountsByTypeAndTopicIDs(ctx, commentTypeTalk, ids)
	if err != nil {
		slog.WarnContext(ctx, "load discovery comment counts failed", "error", err)
		return
	}
	byTopic := make(map[int]int, len(counts))
	for _, count := range counts {
		if count != nil {
			byTopic[count.Id] = count.CommentCount
		}
	}
	for _, talk := range talks {
		if talk == nil {
			continue
		}
		talk.CommentCount = byTopic[talk.Id]
	}
}
func (s *MyPlatformService) Dashboard(c *gin.Context) model.ResultVO {
	dto, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	dashboard, err := s.platformRepo().StudioDashboard(c.Request.Context(), dto.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(dashboard)
}

func (s *MyPlatformService) SyncActivation(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioActivationVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	activation, err := s.platformRepo().SyncStudioActivation(c.Request.Context(), user.UserInfoId, port.StudioActivationUpdate{
		Started: vo.Started, Collapsed: vo.Collapsed, IdentityComplete: vo.IdentityComplete,
		ContentComplete: vo.ContentComplete, ProfileVisited: vo.ProfileVisited, Completed: vo.Completed,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(activation)
}

func (s *MyPlatformService) GetProfile(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	profile, err := s.platformRepo().GetStudioProfile(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(studioProfileDTO(profile))
}

func (s *MyPlatformService) UpdateProfile(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioProfileVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	profile, message := normalizeStudioProfile(vo)
	if message != "" {
		return model.ResultFailWithMessage(message)
	}
	if err := s.platformRepo().UpdateAuthorProfile(c.Request.Context(), user.UserInfoId, profile.Handle, profile.Nickname, profile.Intro, profile.Website); err != nil {
		if apperrors.KindOf(err) == apperrors.KindConflict {
			return model.ResultFailWithMessage("该 Handle 已被占用")
		}
		return model.ResultFromError(err)
	}
	saved, err := s.platformRepo().GetStudioProfile(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(studioProfileDTO(saved))
}

func normalizeStudioProfile(vo model.StudioProfileVO) (model.StudioProfileVO, string) {
	vo.Handle = strings.ToLower(strings.TrimSpace(vo.Handle))
	vo.Nickname = strings.TrimSpace(vo.Nickname)
	vo.Intro = strings.TrimSpace(vo.Intro)
	vo.Website = strings.TrimSpace(vo.Website)
	if !validStudioHandle(vo.Handle) {
		return vo, "Handle 需为 3-40 位小写字母、数字或连字符，且必须以字母或数字开头"
	}
	if vo.Nickname == "" {
		return vo, "昵称不能为空"
	}
	if len([]rune(vo.Nickname)) > 30 {
		return vo, "昵称不能超过 30 个字"
	}
	if len([]rune(vo.Intro)) > 255 {
		return vo, "个人简介不能超过 255 个字"
	}
	if len([]rune(vo.Website)) > 255 {
		return vo, "个人网站不能超过 255 个字"
	}
	if vo.Website != "" && !validStudioWebsite(vo.Website) {
		return vo, "个人网站必须是有效的 HTTP(S) 地址"
	}
	return vo, ""
}

func validStudioHandle(value string) bool {
	if len(value) < 3 || len(value) > 40 {
		return false
	}
	for index, r := range value {
		if index == 0 && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func validStudioWebsite(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func studioProfileDTO(profile port.StudioProfile) model.StudioProfileDTO {
	return model.StudioProfileDTO{
		Handle: profile.Handle, Nickname: profile.Nickname, Avatar: profile.Avatar,
		Intro: profile.Intro, Website: profile.Website,
	}
}
func (s *MyPlatformService) ListOwnedArticles(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	filter, err := studioFilter(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	articles, count, err := s.platformRepo().ListOwnedArticles(c.Request.Context(), user.UserInfoId, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: articles, Count: count})
}

func (s *MyPlatformService) GetOwnedArticle(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "articleId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	article, err := s.platformRepo().GetOwnedArticle(c.Request.Context(), user.UserInfoId, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(article)
}

func (s *MyPlatformService) SaveOwnedArticle(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioArticleVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	status, ok := visibilityStatus(vo.Visibility, true)
	if !ok {
		return model.ResultFailWithMessage("可见性参数不正确")
	}
	title := strings.TrimSpace(vo.ArticleTitle)
	if title == "" {
		return model.ResultFailWithMessage("文章标题不能为空")
	}
	if len([]rune(title)) > 50 {
		return model.ResultFailWithMessage("文章标题不能超过 50 字")
	}
	if len([]rune(vo.ArticleCover)) > 1024 {
		return model.ResultFailWithMessage("文章封面地址过长")
	}
	if len([]rune(vo.OriginalUrl)) > 255 {
		return model.ResultFailWithMessage("原文链接不能超过 255 字")
	}
	if len([]rune(vo.Password)) > 255 {
		return model.ResultFailWithMessage("访问密码不能超过 255 字")
	}
	if vo.Type != 0 && (vo.Type < 1 || vo.Type > 3) {
		return model.ResultFailWithMessage("文章类型不正确")
	}
	if strings.TrimSpace(vo.ArticleContentHTML) != "" {
		vo.ArticleContentHTML = sanitizeArticleHTML(vo.ArticleContentHTML)
		if !articleHTMLHasContent(vo.ArticleContentHTML) {
			return model.ResultFailWithMessage("文章内容不能为空")
		}
		vo.ArticleContent = vo.ArticleContentHTML
	} else if strings.TrimSpace(vo.ArticleContent) == "" {
		return model.ResultFailWithMessage("文章内容不能为空")
	}
	if len([]rune(vo.ArticleContent)) > 100000 {
		return model.ResultFailWithMessage("文章内容不能超过 100000 字")
	}
	article := entity.TArticle{
		Id: vo.Id, UserId: user.UserInfoId, ArticleCover: strings.TrimSpace(vo.ArticleCover), ArticleTitle: title,
		ArticleContent: vo.ArticleContent, ArticleContentHTML: vo.ArticleContentHTML, SeriesId: vo.SeriesId,
		SeriesOrder: vo.SeriesOrder, Status: status, Type: vo.Type, Password: strings.TrimSpace(vo.Password), OriginalUrl: strings.TrimSpace(vo.OriginalUrl),
	}
	if article.Type == 0 {
		article.Type = 1
	}
	if status != 1 {
		article.Password = ""
	}
	if status == 4 {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(vo.ScheduledAt))
		if err != nil || !parsed.After(time.Now()) {
			return model.ResultFailWithMessage("定时发布时间必须晚于当前时间")
		}
		article.ScheduledAt = parsed
	}
	saved, err := s.platformRepo().SaveOwnedArticle(c.Request.Context(), user.UserInfoId, article, vo.CategoryId, vo.TagIds)
	if err != nil {
		return model.ResultFromError(err)
	}
	if s.cache != nil && saved.Id > 0 {
		_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(saved.Id))
	}
	return model.ResultOkWithData(map[string]int{"id": saved.Id})
}

func (s *MyPlatformService) DeleteOwnedArticles(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedArticles(c.Request.Context(), user.UserInfoId, ids); err != nil {
		return model.ResultFromError(err)
	}
	for _, id := range ids {
		if s.cache != nil {
			_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(id))
		}
	}
	return model.ResultOk()
}

func (s *MyPlatformService) Upload(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	kind := strings.TrimSpace(c.Query("kind"))
	prefix := "studio/articles/"
	switch kind {
	case "article-cover", "cover":
		prefix = "articles/covers/"
	case "article-inline":
		prefix = "articles/inline/"
	case "talk-image", "talk":
		prefix = "talks/"
	case "series-cover":
		prefix = "series/covers/"
	case "avatar":
		prefix = "avatar/"
	}
	ref, err := uploadMultipart(c.Request.Context(), s.storage, file, prefix)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(ref.URL)
}

func (s *MyPlatformService) ListOwnedTalks(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	filter, err := studioFilter(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	talks, count, err := s.platformRepo().ListOwnedTalks(c.Request.Context(), user.UserInfoId, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: count})
}

func (s *MyPlatformService) GetOwnedTalk(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "talkId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	talk, err := s.platformRepo().GetOwnedTalk(c.Request.Context(), user.UserInfoId, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(talk)
}

func (s *MyPlatformService) SaveOwnedTalk(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioTalkVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	status, ok := visibilityStatus(vo.Visibility, false)
	if !ok {
		return model.ResultFailWithMessage("可见性参数不正确")
	}
	content := strings.TrimSpace(vo.Content)
	if content == "" {
		return model.ResultFailWithMessage("说说内容不能为空")
	}
	if len([]rune(content)) > 2000 {
		return model.ResultFailWithMessage("说说内容不能超过 2000 字")
	}
	images, err := normalizeTalkImages(strings.TrimSpace(vo.Images))
	if err != nil {
		return model.ResultFailWithMessage("说说图片格式不正确")
	}
	if len([]rune(images)) > 2500 {
		return model.ResultFailWithMessage("说说图片地址过长")
	}
	talk, err := s.platformRepo().SaveOwnedTalk(c.Request.Context(), entity.TTalk{
		Id: vo.Id, UserId: user.UserInfoId, Content: content, Images: images,
		IsTop: vo.IsTop, Status: status,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(map[string]int{"id": talk.Id})
}

func (s *MyPlatformService) DeleteOwnedTalks(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedTalks(c.Request.Context(), user.UserInfoId, ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyPlatformService) ListOwnedSeries(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	filter, err := studioFilter(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	series, count, err := s.platformRepo().ListOwnedSeries(c.Request.Context(), user.UserInfoId, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: series, Count: count})
}

func (s *MyPlatformService) GetOwnedSeries(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "seriesId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	series, err := s.platformRepo().GetOwnedSeries(c.Request.Context(), user.UserInfoId, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(series)
}

func (s *MyPlatformService) SaveOwnedSeries(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioSeriesVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	status, ok := visibilityStatus(vo.Visibility, false)
	if !ok {
		return model.ResultFailWithMessage("可见性参数不正确")
	}
	name := strings.TrimSpace(vo.SeriesName)
	if name == "" {
		return model.ResultFailWithMessage("系列名称不能为空")
	}
	if len([]rune(name)) > 50 {
		return model.ResultFailWithMessage("系列名称不能超过 50 字")
	}
	description := strings.TrimSpace(vo.SeriesDesc)
	if len([]rune(description)) > 255 {
		return model.ResultFailWithMessage("系列简介不能超过 255 字")
	}
	if len([]rune(vo.Cover)) > 1024 {
		return model.ResultFailWithMessage("系列封面地址过长")
	}
	series, err := s.platformRepo().SaveOwnedSeries(c.Request.Context(), entity.TSeries{
		Id: vo.Id, UserId: user.UserInfoId, SeriesName: name,
		SeriesDesc: description, Cover: strings.TrimSpace(vo.Cover), Status: status,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(map[string]int{"id": series.Id})
}

func (s *MyPlatformService) DeleteOwnedSeries(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "seriesId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedSeries(c.Request.Context(), user.UserInfoId, id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyPlatformService) BatchPreviewContent(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioBatchPreviewVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	contentType, ok := studioBatchContentType(vo.Kind)
	if !ok {
		return model.ResultFailWithMessage("内容类型不正确")
	}
	if !validStudioStatus(contentType, vo.Status) {
		return model.ResultFailWithMessage("状态筛选不正确")
	}
	preview, err := s.platformRepo().PreviewOwnedContent(c.Request.Context(), user.UserInfoId, contentType, port.StudioFilter{
		Status: vo.Status, Keywords: strings.TrimSpace(vo.Keywords), SeriesID: vo.SeriesId,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(preview)
}

func (s *MyPlatformService) BatchUpdateContentStatus(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioBatchStatusVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	contentType, ok := studioBatchContentType(vo.Kind)
	if !ok {
		return model.ResultFailWithMessage("内容类型不正确")
	}
	scope, ok := studioBatchScope(vo.Scope, contentType)
	if !ok {
		return model.ResultFailWithMessage("批量选择范围不正确")
	}
	status, ok := visibilityStatus(vo.Visibility, false)
	if !ok {
		return model.ResultFailWithMessage("批量状态仅支持公开、私有或草稿")
	}
	mutation, err := s.platformRepo().BatchUpdateOwnedContentStatus(c.Request.Context(), user.UserInfoId, contentType, scope, status, studioAuditActor(c, user))
	if err != nil {
		return model.ResultFromError(err)
	}
	if contentType == port.StudioContentArticle && s.cache != nil {
		for _, id := range mutation.ContentIDs {
			_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(id))
		}
	}
	return model.ResultOkWithData(mutation)
}

func (s *MyPlatformService) BatchDeleteContent(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioBatchDeleteVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	contentType, ok := studioBatchContentType(vo.Kind)
	if !ok {
		return model.ResultFailWithMessage("内容类型不正确")
	}
	scope, ok := studioBatchScope(vo.Scope, contentType)
	if !ok {
		return model.ResultFailWithMessage("批量选择范围不正确")
	}
	mutation, err := s.platformRepo().BatchDeleteOwnedContent(c.Request.Context(), user.UserInfoId, contentType, scope, studioAuditActor(c, user))
	if err != nil {
		return model.ResultFromError(err)
	}
	if contentType == port.StudioContentArticle && s.cache != nil {
		for _, id := range mutation.ContentIDs {
			_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(id))
		}
	}
	return model.ResultOkWithData(mutation)
}

func (s *MyPlatformService) RetryScheduledPublication(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	articleID, err := pathID(c, "articleId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	record, err := s.platformRepo().RetryScheduledPublication(c.Request.Context(), user.UserInfoId, articleID, studioAuditActor(c, user))
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(record)
}

func optionalUserID(c *gin.Context) int {
	if user, ok := currentUser(c); ok {
		return user.UserInfoId
	}
	return 0
}
func studioAuditActor(c *gin.Context, user model.UserDetailsDTO) port.StudioAuditActor {
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = user.IpAddress
	}
	return port.StudioAuditActor{UserID: user.UserInfoId, Nickname: user.Nickname, IPAddress: ip, IPSource: user.IpSource}
}

func studioBatchContentType(value string) (port.StudioContentType, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(port.StudioContentArticle):
		return port.StudioContentArticle, true
	case string(port.StudioContentTalk):
		return port.StudioContentTalk, true
	case string(port.StudioContentSeries):
		return port.StudioContentSeries, true
	default:
		return "", false
	}
}

func studioBatchScope(vo model.StudioBatchScopeVO, contentType port.StudioContentType) (port.StudioBatchScope, bool) {
	mode := strings.ToLower(strings.TrimSpace(vo.Mode))
	scope := port.StudioBatchScope{
		Mode: mode, Status: vo.Status, Keywords: strings.TrimSpace(vo.Keywords), SeriesID: vo.SeriesId,
		MaxID: vo.MaxId, ExcludeIDs: vo.ExcludeIds, ExpectedCount: vo.ExpectedCount,
	}
	switch mode {
	case port.StudioBatchScopeIDs:
		ids, ok := studioBatchIDs(vo.Ids, 500)
		if !ok {
			return port.StudioBatchScope{}, false
		}
		scope.IDs = ids
		return scope, true
	case port.StudioBatchScopeFilter:
		if scope.ExpectedCount <= 0 || scope.MaxID <= 0 || !validStudioStatus(contentType, scope.Status) {
			return port.StudioBatchScope{}, false
		}
		excludes, ok := studioBatchIDs(vo.ExcludeIds, 500)
		if len(vo.ExcludeIds) > 0 && !ok {
			return port.StudioBatchScope{}, false
		}
		scope.ExcludeIDs = excludes
		if contentType != port.StudioContentArticle && scope.SeriesID != 0 {
			return port.StudioBatchScope{}, false
		}
		return scope, true
	default:
		return port.StudioBatchScope{}, false
	}
}

func validStudioStatus(contentType port.StudioContentType, status int) bool {
	if status < 0 {
		return false
	}
	if contentType == port.StudioContentArticle {
		return status <= 4
	}
	return status <= 3
}

func studioBatchIDs(values []int, limit int) ([]int, bool) {
	if len(values) == 0 {
		return []int{}, true
	}
	if limit <= 0 || len(values) > limit {
		return nil, false
	}
	seen := make(map[int]struct{}, len(values))
	ids := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			return nil, false
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		return nil, false
	}
	return ids, true
}

func studioScopeIDs(scope port.StudioBatchScope) []int {
	if scope.Mode == port.StudioBatchScopeIDs {
		return scope.IDs
	}
	return nil
}
func (s *MyPlatformService) ListOwnedCategories(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	categories, err := s.platformRepo().ListOwnedCategories(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(categories)
}

func (s *MyPlatformService) SaveOwnedCategory(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioTaxonomyVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	category, err := s.platformRepo().SaveOwnedCategory(c.Request.Context(), entity.TCategory{
		Id: vo.Id, UserId: user.UserInfoId, CategoryName: strings.TrimSpace(vo.Name),
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(category)
}

func (s *MyPlatformService) DeleteOwnedCategory(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "categoryId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedCategory(c.Request.Context(), user.UserInfoId, id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyPlatformService) ListOwnedTags(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	tags, err := s.platformRepo().ListOwnedTags(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(tags)
}

func (s *MyPlatformService) SaveOwnedTag(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioTaxonomyVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	tag, err := s.platformRepo().SaveOwnedTag(c.Request.Context(), entity.TTag{
		Id: vo.Id, UserId: user.UserInfoId, TagName: strings.TrimSpace(vo.Name),
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(tag)
}

func (s *MyPlatformService) DeleteOwnedTag(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "tagId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedTag(c.Request.Context(), user.UserInfoId, id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyPlatformService) Moderate(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.ModerationVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if vo.Id <= 0 {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	reason := strings.TrimSpace(vo.Reason)
	if vo.Hidden && reason == "" {
		return model.ResultFailWithMessage("隐藏内容时必须填写审核理由")
	}
	if len([]rune(reason)) > 255 {
		return model.ResultFailWithMessage("审核理由不能超过 255 字")
	}
	if err := s.platformRepo().ModerateContent(c.Request.Context(), strings.TrimSpace(vo.ContentType), vo.Id, user.UserInfoId, vo.Hidden, reason); err != nil {
		return model.ResultFromError(err)
	}
	if s.cache != nil {
		_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(vo.Id))
	}
	return model.ResultOk()
}

func (s *MyPlatformService) DistributeArticle(c *gin.Context) model.ResultVO {
	id, err := pathID(c, "articleId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.DistributionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	current, err := s.articles.GetArticleRecord(c.Request.Context(), id)
	if err != nil {
		return model.ResultFromError(err)
	}
	if current.IsDelete != 0 || current.Status != 1 || current.ModerationStatus != "visible" {
		return model.ResultFailWithMessage("只有公开且审核可见的文章可以推荐或分发")
	}
	if vo.Newsletter && s.newsletter == nil {
		return model.ResultFromError(apperrors.Unavailable("article.distribution", nil))
	}
	featured := 0
	if vo.Featured {
		featured = 1
	}
	article, err := s.articles.UpdateTopAndFeatured(c.Request.Context(), id, 0, featured)
	if err != nil {
		return model.ResultFromError(err)
	}
	if article.Id == 0 {
		return model.ResultFailWithMessage("文章不存在")
	}
	if vo.Newsletter {
		if err := s.newsletter.EnqueueArticle(c.Request.Context(), id); err != nil {
			return model.ResultFromError(err)
		}
	}
	if s.cache != nil {
		_ = s.cache.Delete(c.Request.Context(), strconv.Itoa(id))
	}
	return model.ResultOk()
}

func currentUser(c *gin.Context) (model.UserDetailsDTO, bool) {
	value, ok := c.Get("userInfo")
	if !ok {
		return model.UserDetailsDTO{}, false
	}
	user, ok := value.(model.UserDetailsDTO)
	if !ok || user.UserInfoId <= 0 {
		return model.UserDetailsDTO{}, false
	}
	return user, true
}

func pageParams(c *gin.Context) (int, int, error) {
	current, err := strconv.Atoi(c.DefaultQuery("current", "1"))
	if err != nil || current < 1 {
		return 0, 0, errInvalidPage
	}
	size, err := strconv.Atoi(c.DefaultQuery("size", "12"))
	if err != nil || size < 1 {
		return 0, 0, errInvalidPage
	}
	if size > 100 {
		size = 100
	}
	return current, size, nil
}

func studioFilter(c *gin.Context) (port.StudioFilter, error) {
	current, size, err := pageParams(c)
	if err != nil {
		return port.StudioFilter{}, err
	}
	status := 0
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status, err = strconv.Atoi(raw)
		if err != nil {
			return port.StudioFilter{}, errInvalidPage
		}
	}
	seriesID := 0
	if raw := strings.TrimSpace(c.Query("seriesId")); raw != "" {
		seriesID, err = strconv.Atoi(raw)
		if err != nil || seriesID <= 0 {
			return port.StudioFilter{}, errInvalidPage
		}
	}
	return port.StudioFilter{Current: current, Size: size, Status: status, SeriesID: seriesID, Keywords: c.Query("keywords")}, nil
}

func pathID(c *gin.Context, name string) (int, error) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		return 0, errInvalidPage
	}
	return id, nil
}

func visibilityStatus(value string, article bool) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "public":
		return 1, true
	case "private":
		return 2, true
	case "draft":
		return 3, true
	case "scheduled":
		return 4, article
	default:
		return 0, false
	}
}

var errInvalidPage = &strconv.NumError{Func: "page", Num: "", Err: strconv.ErrSyntax}
