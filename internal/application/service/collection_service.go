package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

const (
	collectionMaxTitle       = 80
	collectionMaxDescription = 500
	collectionMaxNote        = 280
	collectionMaxPerUser     = 50
	collectionMaxItems       = 500
)

type CollectionService interface {
	ListPublic(c *gin.Context) model.ResultVO
	GetPublic(c *gin.Context) model.ResultVO
	ListAuthor(c *gin.Context) model.ResultVO
	ListOwned(c *gin.Context) model.ResultVO
	GetOwned(c *gin.Context) model.ResultVO
	Create(c *gin.Context) model.ResultVO
	Update(c *gin.Context) model.ResultVO
	Delete(c *gin.Context) model.ResultVO
	AddItem(c *gin.Context) model.ResultVO
	RemoveItem(c *gin.Context) model.ResultVO
	Reorder(c *gin.Context) model.ResultVO
	ListAdmin(c *gin.Context) model.ResultVO
}

type MyCollectionService struct {
	collections port.CollectionRepository
	platform    port.PlatformRepository
	articles    port.ArticleRepository
	reactions   port.ArticleReactionRepository
}

func NewCollectionService(collections port.CollectionRepository, platform port.PlatformRepository, articles port.ArticleRepository, reactions port.ArticleReactionRepository) (*MyCollectionService, error) {
	if collections == nil || platform == nil || articles == nil || reactions == nil {
		return nil, missingServiceDependency("collection", "repository")
	}
	return &MyCollectionService{collections: collections, platform: platform, articles: articles, reactions: reactions}, nil
}

func collectionSlug(title string) string {
	var builder strings.Builder
	lastDash := false
	for _, value := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case value >= 'a' && value <= 'z', value >= '0' && value <= '9':
			builder.WriteRune(value)
			lastDash = false
		case value == '-' || value == '_' || value == ' ':
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
		if builder.Len() >= 40 {
			break
		}
	}
	base := strings.Trim(builder.String(), "-")
	randomBytes := make([]byte, 5)
	if _, err := rand.Read(randomBytes); err != nil {
		randomBytes = []byte{0, 1, 2, 3, 4}
	}
	suffix := hex.EncodeToString(randomBytes)
	if base == "" {
		return "list-" + suffix
	}
	return base + "-" + suffix
}

func normalizedCollectionSave(vo model.CollectionSaveVO) (port.CollectionSaveInput, error) {
	title := strings.TrimSpace(vo.Title)
	description := strings.TrimSpace(vo.Description)
	visibility := strings.ToLower(strings.TrimSpace(vo.Visibility))
	if title == "" || utf8.RuneCountInString(title) > collectionMaxTitle {
		return port.CollectionSaveInput{}, apperrors.Invalid("collection.title", "invalid title")
	}
	if utf8.RuneCountInString(description) > collectionMaxDescription {
		return port.CollectionSaveInput{}, apperrors.Invalid("collection.description", "description is too long")
	}
	if !port.ValidCollectionVisibility(visibility) {
		return port.CollectionSaveInput{}, apperrors.Invalid("collection.visibility", "invalid visibility")
	}
	return port.CollectionSaveInput{Title: title, Description: description, Visibility: visibility}, nil
}

func (s *MyCollectionService) attachCollectionItems(ctx context.Context, record port.CollectionRecord, includeUnavailable bool) (port.CollectionDetail, error) {
	ids := make([]int, 0, len(record.Items))
	for _, item := range record.Items {
		if item.Available || includeUnavailable {
			ids = append(ids, item.ArticleID)
		}
	}
	cards, err := s.articles.ListArticleCardsByIDs(ctx, ids)
	if err != nil {
		return port.CollectionDetail{}, err
	}
	cardByID := make(map[int]*port.ArticleCard, len(cards))
	for _, card := range cards {
		cardByID[card.Id] = card
	}
	if len(cards) > 0 {
		cardIDs := make([]int, 0, len(cards))
		for _, card := range cards {
			cardIDs = append(cardIDs, card.Id)
		}
		counts, countErr := s.reactions.Counts(ctx, cardIDs)
		if countErr == nil {
			for _, card := range cards {
				value := counts[card.Id]
				card.LikeCount = value.LikeCount
				card.FavoriteCount = value.FavoriteCount
			}
		}
	}
	items := make([]port.CollectionItemView, 0, len(record.Items))
	for _, item := range record.Items {
		card := cardByID[item.ArticleID]
		if card == nil && !includeUnavailable {
			continue
		}
		items = append(items, port.CollectionItemView{
			ArticleID: item.ArticleID, Note: item.Note, Position: item.Position,
			Available: item.Available, Article: card,
		})
	}
	return port.CollectionDetail{Collection: record.Collection, Items: items}, nil
}

func (s *MyCollectionService) ListPublic(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	sort := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	if sort != port.CollectionSortHot && sort != port.CollectionSortLatest {
		sort = port.CollectionSortLatest
	}
	items, total, err := s.collections.ListPublic(c.Request.Context(), sort, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items, Count: total, Page: current, PageSize: size})
}

func (s *MyCollectionService) GetPublic(c *gin.Context) model.ResultVO {
	record, err := s.collections.GetPublicBySlug(c.Request.Context(), strings.TrimSpace(c.Param("slug")))
	if err != nil {
		return model.ResultFromError(err)
	}
	detail, err := s.attachCollectionItems(c.Request.Context(), record, false)
	if err != nil {
		return model.ResultFromError(err)
	}
	if record.Collection.Visibility == port.CollectionVisibilityPublic && len(detail.Items) == 0 {
		return model.ResultFromError(apperrors.NotFound("collection.public.get"))
	}
	return model.ResultOkWithData(detail)
}

func (s *MyCollectionService) ListAuthor(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	author, err := s.platform.GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	items, total, err := s.collections.ListPublicByOwner(c.Request.Context(), author.Id, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items, Count: total, Page: current, PageSize: size})
}

func (s *MyCollectionService) ListOwned(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	items, total, err := s.collections.ListOwned(c.Request.Context(), user.UserInfoId, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items, Count: total, Page: current, PageSize: size})
}

func (s *MyCollectionService) GetOwned(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	record, err := s.collections.GetOwned(c.Request.Context(), user.UserInfoId, id)
	if err != nil {
		return model.ResultFromError(err)
	}
	detail, err := s.attachCollectionItems(c.Request.Context(), record, true)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(detail)
}

func (s *MyCollectionService) Create(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.CollectionSaveVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	input, err := normalizedCollectionSave(vo)
	if err != nil {
		return model.ResultFromError(err)
	}
	_, total, err := s.collections.ListOwned(c.Request.Context(), user.UserInfoId, 1, collectionMaxPerUser+1)
	if err != nil {
		return model.ResultFromError(err)
	}
	if total >= collectionMaxPerUser {
		return model.ResultFailWithMessage("收藏集数量已达上限")
	}
	item, err := s.collections.CreateOwned(c.Request.Context(), user.UserInfoId, collectionSlug(input.Title), input)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(item)
}

func (s *MyCollectionService) Update(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.CollectionSaveVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	input, err := normalizedCollectionSave(vo)
	if err != nil {
		return model.ResultFromError(err)
	}
	item, err := s.collections.UpdateOwned(c.Request.Context(), user.UserInfoId, id, input)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(item)
}

func (s *MyCollectionService) Delete(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.collections.DeleteOwned(c.Request.Context(), user.UserInfoId, id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyCollectionService) AddItem(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	articleID, err := pathID(c, "articleId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.CollectionItemVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	note := strings.TrimSpace(vo.Note)
	if utf8.RuneCountInString(note) > collectionMaxNote {
		return model.ResultFailWithMessage("推荐语不能超过 280 字")
	}
	article, err := s.articles.GetArticleRecord(c.Request.Context(), articleID)
	if err != nil {
		return model.ResultFromError(err)
	}
	if article.Id == 0 || article.IsDelete != 0 || article.Status != 1 || article.ModerationStatus != "visible" {
		return model.ResultFromError(apperrors.NotFound("collection.item.article"))
	}
	collection, err := s.collections.GetOwned(c.Request.Context(), user.UserInfoId, collectionID)
	if err != nil {
		return model.ResultFromError(err)
	}
	if len(collection.Items) >= collectionMaxItems {
		found := false
		for _, item := range collection.Items {
			if item.ArticleID == articleID {
				found = true
				break
			}
		}
		if !found {
			return model.ResultFailWithMessage("收藏集文章数量已达上限")
		}
	}
	if err := s.collections.AddItem(c.Request.Context(), user.UserInfoId, collectionID, articleID, note); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyCollectionService) RemoveItem(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	articleID, err := pathID(c, "articleId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.collections.RemoveItem(c.Request.Context(), user.UserInfoId, collectionID, articleID); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyCollectionService) Reorder(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(c, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.CollectionOrderVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	seen := map[int]struct{}{}
	order := make([]int, 0, len(vo.ArticleIDs))
	for _, articleID := range vo.ArticleIDs {
		if articleID <= 0 {
			return model.ResultFailWithMessage("参数格式不正确")
		}
		if _, exists := seen[articleID]; exists {
			continue
		}
		seen[articleID] = struct{}{}
		order = append(order, articleID)
	}
	if err := s.collections.Reorder(c.Request.Context(), user.UserInfoId, collectionID, order); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyCollectionService) ListAdmin(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	items, total, err := s.collections.ListAdmin(c.Request.Context(), current, size, strings.TrimSpace(c.Query("moderation")), c.Query("keywords"))
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items, Count: total, Page: current, PageSize: size})
}
