package service

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

// ArticleAdminUseCases is the typed application boundary for admin article
// operations. Transport binding, authentication extraction and response
// envelopes belong to the HTTP adapter.
type ArticleAdminUseCases interface {
	ListAdminArticles(context.Context, port.ArticleFilter) (ArticlePage[*port.ArticleAdmin], error)
	SaveAdminArticle(context.Context, ArticleSaveInput) error
	SetArticleTopAndFeatured(context.Context, int, int, int) error
	TrashArticles(context.Context, int, []int, int) error
	DeleteArticles(context.Context, int, []int) error
	UploadArticleImage(context.Context, ArticleImageUpload) (port.ObjectRef, error)
	GetAdminArticle(context.Context, int) (port.ArticleAdminView, bool, error)
	ImportAdminArticle(context.Context, ArticleImportInput) error
	ExportAdminArticles(context.Context, []int) ([]string, error)
}

// ArticleSaveInput contains application-level article data independent of an
// HTTP request or the wire DTO used to populate it.
type ArticleSaveInput struct {
	Article      entity.TArticle
	CategoryName string
	TagNames     []string
	ScheduledAt  string
	UserID       int
}

type ArticleImageUpload struct {
	Filename    string
	ContentType string
	Size        int64
	Content     io.Reader
}

type ArticleImportInput struct {
	Filename string
	Content  io.Reader
	UserID   int
}

type ArticleAdminFailure string

const (
	ArticleAdminContentRequired ArticleAdminFailure = "content_required"
	ArticleAdminPublicRequired  ArticleAdminFailure = "public_article_required"
	ArticleAdminImportRead      ArticleAdminFailure = "import_read_failed"
)

// ArticleAdminError identifies the few legacy failures with messages that are
// more specific than the shared domain error mapping.
type ArticleAdminError struct {
	Failure ArticleAdminFailure
}

func (e *ArticleAdminError) Error() string {
	if e == nil {
		return "article admin operation failed"
	}
	return string(e.Failure)
}

func (a *MyArticleService) ListAdminArticles(ctx context.Context, filter port.ArticleFilter) (ArticlePage[*port.ArticleAdmin], error) {
	total, err := a.repo.CountArticleAdmins(ctx, filter)
	if err != nil {
		return ArticlePage[*port.ArticleAdmin]{}, err
	}
	items, err := a.repo.ListArticlesAdmin(ctx, filter)
	if err != nil {
		return ArticlePage[*port.ArticleAdmin]{}, err
	}
	if items == nil {
		items = []*port.ArticleAdmin{}
	}
	if a.cache != nil {
		views, viewErr := a.cache.ZRangeWithScores(ctx, ArticleViewsCount)
		if viewErr != nil {
			slog.WarnContext(ctx, "load article view counts failed", "error", viewErr)
		} else {
			for _, item := range items {
				if item == nil {
					continue
				}
				if count := views[strconv.Itoa(item.Id)]; count != 0 {
					item.ViewsCount = int(count)
				}
			}
		}
	}
	a.attachAdminReactionCounts(ctx, items)
	return ArticlePage[*port.ArticleAdmin]{Items: items, Total: total, Page: filter.Current, PageSize: filter.Size}, nil
}

func (a *MyArticleService) SaveAdminArticle(ctx context.Context, input ArticleSaveInput) error {
	article := input.Article
	article.UserId = input.UserID
	if strings.TrimSpace(article.ArticleContentHTML) != "" {
		article.ArticleContentHTML = sanitizeArticleHTML(article.ArticleContentHTML)
		if !articleHTMLHasContent(article.ArticleContentHTML) {
			return &ArticleAdminError{Failure: ArticleAdminContentRequired}
		}
		// Keep the legacy field populated so older readers and Markdown exports
		// continue to receive a renderable article body.
		article.ArticleContent = article.ArticleContentHTML
	} else if strings.TrimSpace(article.ArticleContent) == "" {
		return &ArticleAdminError{Failure: ArticleAdminContentRequired}
	}
	scheduledAt, err := normalizeScheduledAt(article.Status, input.ScheduledAt)
	if err != nil {
		return err
	}
	article.ScheduledAt = scheduledAt

	previousStatus := 0
	if article.Id != 0 {
		previous, err := a.repo.GetArticleRecord(ctx, article.Id)
		if err != nil {
			return err
		}
		if previous.UserId != input.UserID {
			return apperrors.New(apperrors.KindForbidden, "article.save", nil)
		}
		previousStatus = previous.Status
	}
	saved, err := a.repo.SaveOrUpdate(ctx, article, input.CategoryName, input.TagNames)
	if err != nil {
		return err
	}
	if saved.Id != 0 {
		if a.newsletter != nil && previousStatus != 1 && saved.Status == 1 && saved.IsDelete == 0 {
			if err := a.newsletter.EnqueueArticle(ctx, saved.Id); err != nil {
				// Publishing must not fail because the notification outbox is
				// temporarily unavailable; an admin can re-enqueue later.
				slog.ErrorContext(ctx, "enqueue newsletter article failed", "articleId", saved.Id, "error", err)
			}
		}
		a.cacheArticle(ctx, saved.Id, saved.Status, saved.IsDelete, saved)
		a.syncArticleSearch(ctx, saved.Id)
	}
	return nil
}

func (a *MyArticleService) SetArticleTopAndFeatured(ctx context.Context, articleID, isTop, isFeatured int) error {
	if isTop == 1 || isFeatured == 1 {
		current, err := a.repo.GetArticleRecord(ctx, articleID)
		if err != nil {
			return err
		}
		if current.IsDelete != 0 || current.Status != 1 || current.ModerationStatus != "visible" {
			return &ArticleAdminError{Failure: ArticleAdminPublicRequired}
		}
	}
	article, err := a.repo.UpdateTopAndFeatured(ctx, articleID, isTop, isFeatured)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return nil
		}
		return err
	}
	if article.Id != 0 {
		a.cacheArticle(ctx, article.Id, article.Status, article.IsDelete, article)
		a.syncArticleSearch(ctx, article.Id)
	}
	return nil
}

func (a *MyArticleService) TrashArticles(ctx context.Context, userID int, ids []int, isDelete int) error {
	if err := a.requireOwnedArticles(ctx, userID, ids); err != nil {
		return err
	}
	if err := a.repo.UpdateDelete(ctx, ids, isDelete); err != nil {
		return err
	}
	for _, id := range ids {
		a.evictArticleCache(ctx, strconv.Itoa(id))
	}
	a.syncArticleSearch(ctx, ids...)
	return nil
}

func (a *MyArticleService) DeleteArticles(ctx context.Context, userID int, ids []int) error {
	if err := a.requireOwnedArticles(ctx, userID, ids); err != nil {
		return err
	}
	if err := a.repo.Delete(ctx, ids); err != nil {
		return err
	}
	a.syncArticleSearch(ctx, ids...)
	return nil
}

func (a *MyArticleService) requireOwnedArticles(ctx context.Context, userID int, ids []int) error {
	if userID <= 0 {
		return apperrors.New(apperrors.KindUnauthorized, "article.owner", nil)
	}
	if len(ids) == 0 {
		return apperrors.Invalid("article.owner", "article id is required")
	}
	for _, id := range ids {
		if id <= 0 {
			return apperrors.Invalid("article.owner", "article id is invalid")
		}
		article, err := a.repo.GetArticleRecord(ctx, id)
		if err != nil {
			return err
		}
		if article.UserId != userID {
			return apperrors.New(apperrors.KindForbidden, "article.owner", nil)
		}
	}
	return nil
}

func (a *MyArticleService) UploadArticleImage(ctx context.Context, upload ArticleImageUpload) (port.ObjectRef, error) {
	if a.storage == nil {
		return port.ObjectRef{}, apperrors.Unavailable("storage.upload", nil)
	}
	if upload.Size > maxImageUploadBytes {
		return port.ObjectRef{}, apperrors.Invalid("storage.upload", "image file is too large")
	}
	contentType := strings.ToLower(strings.TrimSpace(upload.ContentType))
	if !strings.HasPrefix(contentType, "image/") {
		contentType = strings.ToLower(strings.TrimSpace(mime.TypeByExtension(filepath.Ext(upload.Filename))))
	}
	if !strings.HasPrefix(contentType, "image/") {
		return port.ObjectRef{}, apperrors.Invalid("storage.upload", "only image files are supported")
	}
	if upload.Content == nil {
		return port.ObjectRef{}, apperrors.Invalid("storage.upload", "file is required")
	}
	key, err := ObjectKey(upload.Filename, "articles/")
	if err != nil {
		return port.ObjectRef{}, apperrors.Invalid("storage.upload", err.Error())
	}
	return a.storage.Put(ctx, key, upload.Content)
}

// normalizeScheduledAt keeps scheduled releases consistent with status 4:
// scheduled articles require a future timestamp, while other states clear it.
func normalizeScheduledAt(status int, raw string) (time.Time, error) {
	if status != ArticleStatusScheduled {
		return time.Time{}, nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, apperrors.Invalid("article.schedule", "scheduled articles need a release time")
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		parsed, err = time.Parse("2006-01-02 15:04:05", raw)
	}
	if err != nil {
		return time.Time{}, apperrors.Invalid("article.schedule", "release time must be an RFC3339 timestamp")
	}
	if !parsed.After(time.Now()) {
		return time.Time{}, apperrors.Invalid("article.schedule", "release time must be in the future")
	}
	return parsed, nil
}

func (a *MyArticleService) GetAdminArticle(ctx context.Context, articleID int) (port.ArticleAdminView, bool, error) {
	article, categoryName, tagNames, err := a.repo.GetAdminArticle(ctx, articleID)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ArticleAdminView{}, false, nil
		}
		return port.ArticleAdminView{}, false, err
	}
	scheduledAt := article.ScheduledAt.Format(time.RFC3339)
	return port.ArticleAdminView{
		Id: article.Id, UserId: article.UserId, ArticleCover: article.ArticleCover,
		ArticleTitle: article.ArticleTitle, ArticleContent: article.ArticleContent,
		ArticleContentHTML: article.ArticleContentHTML, IsTop: article.IsTop,
		IsFeatured: article.IsFeatured, CategoryId: article.CategoryId,
		CategoryName: categoryName, TagNames: tagNames, Status: article.Status,
		Type: article.Type, Password: article.Password, OriginalUrl: article.OriginalUrl,
		SeriesId: article.SeriesId, SeriesOrder: article.SeriesOrder,
		ScheduledAt: scheduledAt, ModerationStatus: article.ModerationStatus,
		ModerationReason: article.ModerationReason,
	}, true, nil
}

func (a *MyArticleService) ImportAdminArticle(ctx context.Context, input ArticleImportInput) error {
	filename := input.Filename
	index := strings.LastIndex(filename, ".")
	if index <= 0 || index == len(filename)-1 {
		return apperrors.Invalid("article.import", "file name needs an extension")
	}
	if input.Content == nil {
		return apperrors.Invalid("article.import", "file is required")
	}
	content, err := io.ReadAll(input.Content)
	if err != nil {
		return &ArticleAdminError{Failure: ArticleAdminImportRead}
	}
	return a.SaveAdminArticle(ctx, ArticleSaveInput{
		Article: entity.TArticle{ArticleTitle: filename[:index], ArticleContent: string(content), Status: 3},
		UserID:  input.UserID,
	})
}

func (a *MyArticleService) ExportAdminArticles(ctx context.Context, ids []int) ([]string, error) {
	articles, err := a.repo.Export(ctx, ids)
	if err != nil {
		return nil, err
	}
	var urls []string
	for _, article := range articles {
		ref, err := uploadNamed(ctx, a.storage, bytes.NewReader([]byte(article.ArticleContent)), article.ArticleTitle+".md", "markdown/")
		if err != nil {
			return nil, err
		}
		urls = append(urls, ref.URL)
	}
	return urls, nil
}

var _ ArticleAdminUseCases = (*MyArticleService)(nil)
