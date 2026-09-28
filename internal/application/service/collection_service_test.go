package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type fakeCollectionRepository struct {
	record          port.CollectionRecord
	ownedTotal      int
	createCalls     int
	addCalls        int
	reorderCalls    int
	lastCreateInput port.CollectionSaveInput
}

func collectionTestContext(t *testing.T, method, target, body string, params gin.Params) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if body == "" {
		c.Request = httptest.NewRequest(method, target, nil)
	} else {
		c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Params = params
	return c
}

func (f *fakeCollectionRepository) ListPublic(context.Context, string, int, int) ([]*port.CollectionSummary, int, error) {
	return nil, 0, nil
}
func (f *fakeCollectionRepository) ListPublicByOwner(context.Context, int, int, int) ([]*port.CollectionSummary, int, error) {
	return nil, 0, nil
}
func (f *fakeCollectionRepository) GetPublicBySlug(context.Context, string) (port.CollectionRecord, error) {
	return f.record, nil
}
func (f *fakeCollectionRepository) ListOwned(context.Context, int, int, int) ([]*port.CollectionSummary, int, error) {
	return nil, f.ownedTotal, nil
}
func (f *fakeCollectionRepository) GetOwned(context.Context, int, int) (port.CollectionRecord, error) {
	return f.record, nil
}
func (f *fakeCollectionRepository) CreateOwned(_ context.Context, _ int, _ string, input port.CollectionSaveInput) (port.CollectionSummary, error) {
	f.createCalls++
	f.lastCreateInput = input
	return port.CollectionSummary{ID: 20, Title: input.Title, Visibility: input.Visibility}, nil
}
func (f *fakeCollectionRepository) UpdateOwned(context.Context, int, int, port.CollectionSaveInput) (port.CollectionSummary, error) {
	return port.CollectionSummary{}, nil
}
func (f *fakeCollectionRepository) DeleteOwned(context.Context, int, int) error { return nil }
func (f *fakeCollectionRepository) AddItem(context.Context, int, int, int, string) error {
	f.addCalls++
	return nil
}
func (f *fakeCollectionRepository) RemoveItem(context.Context, int, int, int) error { return nil }
func (f *fakeCollectionRepository) Reorder(context.Context, int, int, []int) error {
	f.reorderCalls++
	return nil
}
func (f *fakeCollectionRepository) ListAdmin(context.Context, int, int, string, string) ([]*port.CollectionSummary, int, error) {
	return nil, 0, nil
}

type collectionArticles struct {
	fakeArticleRepository
	cards     map[int]*port.ArticleCard
	record    entity.TArticle
	recordErr error
}

func (f *collectionArticles) ListArticleCardsByIDs(_ context.Context, ids []int) ([]*port.ArticleCard, error) {
	cards := make([]*port.ArticleCard, 0, len(ids))
	for _, id := range ids {
		if card := f.cards[id]; card != nil {
			cards = append(cards, card)
		}
	}
	return cards, nil
}

func (f *collectionArticles) GetArticleRecord(context.Context, int) (entity.TArticle, error) {
	return f.record, f.recordErr
}

func mustCollectionService(t *testing.T, repo port.CollectionRepository, articles port.ArticleRepository) *MyCollectionService {
	t.Helper()
	service, err := NewCollectionService(repo, &fakePlatformRepository{}, articles, &fakeArticleReactionRepository{})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestCollectionSlugAndSaveValidation(t *testing.T) {
	slug := collectionSlug("My Cool_List")
	if !strings.HasPrefix(slug, "my-cool-list-") || len(slug) != len("my-cool-list-")+10 {
		t.Fatalf("unexpected collection slug: %q", slug)
	}
	if chinese := collectionSlug("中文书单"); !strings.HasPrefix(chinese, "list-") || len(chinese) != len("list-")+10 {
		t.Fatalf("unexpected fallback slug: %q", chinese)
	}

	if _, err := normalizedCollectionSave(model.CollectionSaveVO{Title: " ", Visibility: port.CollectionVisibilityPrivate}); err == nil {
		t.Fatal("empty collection title must be rejected")
	}
	if _, err := normalizedCollectionSave(model.CollectionSaveVO{Title: "valid", Visibility: "followers"}); err == nil {
		t.Fatal("unsupported visibility must be rejected")
	}
	if _, err := normalizedCollectionSave(model.CollectionSaveVO{Title: "valid", Description: strings.Repeat("a", collectionMaxDescription+1), Visibility: port.CollectionVisibilityPublic}); err == nil {
		t.Fatal("oversized collection description must be rejected")
	}
	input, err := normalizedCollectionSave(model.CollectionSaveVO{Title: "  title  ", Description: " desc ", Visibility: " PUBLIC "})
	if err != nil || input.Title != "title" || input.Description != "desc" || input.Visibility != port.CollectionVisibilityPublic {
		t.Fatalf("unexpected normalized input: %+v err=%v", input, err)
	}
}

func TestCollectionPublicDetailFiltersUnavailableItems(t *testing.T) {
	repo := &fakeCollectionRepository{record: port.CollectionRecord{
		Collection: port.CollectionSummary{ID: 1, Slug: "reading-path", Visibility: port.CollectionVisibilityPublic},
		Items: []port.CollectionItemRecord{
			{ArticleID: 1, Available: true, Position: 1},
			{ArticleID: 2, Available: false, Position: 2},
		},
	}}
	articles := &collectionArticles{cards: map[int]*port.ArticleCard{1: {Id: 1, ArticleTitle: "visible"}}}
	service := mustCollectionService(t, repo, articles)

	result := service.GetPublic(collectionTestContext(t, http.MethodGet, "/v1/public/collections/reading-path", "", gin.Params{{Key: "slug", Value: "reading-path"}}))
	if !result.Flag {
		t.Fatalf("unexpected public detail result: %+v", result)
	}
	detail, ok := result.Data.(port.CollectionDetail)
	if !ok || len(detail.Items) != 1 || detail.Items[0].ArticleID != 1 || detail.Items[0].Article == nil {
		t.Fatalf("unavailable item must be filtered: %+v", result.Data)
	}
}

func TestCollectionCreateEnforcesLimit(t *testing.T) {
	repo := &fakeCollectionRepository{ownedTotal: collectionMaxPerUser}
	service := mustCollectionService(t, repo, &collectionArticles{})
	c := collectionTestContext(t, http.MethodPost, "/v1/studio/collections", `{"title":"new list","visibility":"private"}`, nil)
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})

	result := service.Create(c)
	if result.Flag || repo.createCalls != 0 {
		t.Fatalf("collection limit must block persistence: result=%+v calls=%d", result, repo.createCalls)
	}
}

func TestCollectionAddItemRequiresPublicVisibleArticle(t *testing.T) {
	repo := &fakeCollectionRepository{record: port.CollectionRecord{Collection: port.CollectionSummary{ID: 9}}}
	articles := &collectionArticles{record: entity.TArticle{Id: 3, Status: 2, ModerationStatus: "visible"}}
	service := mustCollectionService(t, repo, articles)
	c := collectionTestContext(t, http.MethodPut, "/v1/studio/collections/9/items/3", `{"note":"later"}`, gin.Params{
		{Key: "collectionId", Value: "9"}, {Key: "articleId", Value: "3"},
	})
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})

	result := service.AddItem(c)
	if result.Flag || repo.addCalls != 0 {
		t.Fatalf("non-public article must not be collected: result=%+v calls=%d", result, repo.addCalls)
	}
}

func TestCollectionServiceRejectsMissingDependencies(t *testing.T) {
	if _, err := NewCollectionService(nil, nil, nil, nil); err == nil {
		t.Fatal("missing collection dependencies must fail")
	}
}
