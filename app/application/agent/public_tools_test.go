package agent

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"testing"
)

type publicToolArticleRepository struct {
	port.ArticleRepository
	article port.Article
}

func (f publicToolArticleRepository) GetArticleByID(context.Context, int) (port.Article, error) {
	return f.article, nil
}

type publicToolSearcher struct {
	hits []port.ArticleSearchHit
}

func (f publicToolSearcher) Search(context.Context, string) ([]port.ArticleSearchHit, error) {
	return f.hits, nil
}

type publicToolCategoryRepository struct {
	port.CategoryRepository
	categories []port.Category
}

func (f publicToolCategoryRepository) List(context.Context) ([]port.Category, error) {
	return f.categories, nil
}

type publicToolTagRepository struct {
	port.TagRepository
	tags []*port.Tag
}

func (f publicToolTagRepository) List(context.Context) ([]*port.Tag, error) {
	return f.tags, nil
}

type publicToolVitals struct{}

func (publicToolVitals) Snapshot(context.Context) (port.AgentVitals, error) {
	return port.AgentVitals{Status: "awake", LifeStage: "growing", Emotion: "calm"}, nil
}

func newPublicToolRegistryForTest(article port.Article, hits []port.ArticleSearchHit) *PublicToolRegistry {
	registry, err := NewPublicToolRegistry(PublicToolDeps{
		Articles:   publicToolArticleRepository{article: article},
		Searcher:   publicToolSearcher{hits: hits},
		Categories: publicToolCategoryRepository{categories: []port.Category{{Id: 1, CategoryName: "Go"}}},
		Tags:       publicToolTagRepository{tags: []*port.Tag{{Id: 2, TagName: "Agent"}}},
		Vitals:     publicToolVitals{},
	})
	if err != nil {
		panic(err)
	}
	return registry
}

func TestPublicToolDefinitionsAreAnExplicitReadOnlyAllowlist(t *testing.T) {
	registry := newPublicToolRegistryForTest(port.Article{Id: 1, Status: port.PublicArticleStatus}, nil)
	definitions := registry.Definitions()
	if len(definitions) != 4 {
		t.Fatalf("Definitions() returned %d tools, want 4", len(definitions))
	}
	for _, definition := range definitions {
		if definition.Name == "" || !json.Valid(definition.Parameters) {
			t.Fatalf("invalid public definition: %#v", definition)
		}
		if strings.Contains(strings.ToLower(definition.Name), "write") || strings.Contains(strings.ToLower(definition.Description), "写入") {
			t.Fatalf("public definition exposes a write capability: %#v", definition)
		}
	}
}

func TestPublicToolRegistryRejectsUnknownTool(t *testing.T) {
	registry := newPublicToolRegistryForTest(port.Article{Id: 1, Status: port.PublicArticleStatus}, nil)
	_, err := registry.Execute(context.Background(), "delete_article", nil)
	if errors.KindOf(err) != errors.KindForbidden {
		t.Fatalf("unknown tool error kind = %q, want forbidden", errors.KindOf(err))
	}
}

func TestPublicToolReadArticleRejectsPrivateArticle(t *testing.T) {
	registry := newPublicToolRegistryForTest(port.Article{Id: 7, Status: 2, ArticleContent: "private"}, nil)
	_, err := registry.Execute(context.Background(), port.PublicToolReadArticle, json.RawMessage(`{"articleId":7}`))
	if errors.KindOf(err) != errors.KindNotFound {
		t.Fatalf("private article error kind = %q, want not_found", errors.KindOf(err))
	}
}

func TestPublicToolSearchFiltersNonPublicHitsAndReturnsCitation(t *testing.T) {
	registry := newPublicToolRegistryForTest(port.Article{Id: 1, Status: port.PublicArticleStatus}, []port.ArticleSearchHit{
		{ArticleSearch: port.ArticleSearch{Id: 1, ArticleTitle: "公开文章", ArticleContent: "可见内容", Status: 1, IsDelete: 0}},
		{ArticleSearch: port.ArticleSearch{Id: 2, ArticleTitle: "草稿", ArticleContent: "不可见", Status: 2, IsDelete: 0}},
	})
	result, err := registry.Execute(context.Background(), port.PublicToolSearchArticles, json.RawMessage(`{"query":"文章"}`))
	if err != nil {
		t.Fatalf("search tool error = %v", err)
	}
	if strings.Contains(result.Content, "草稿") || len(result.Citations) != 1 || result.Citations[0].ArticleID != 1 {
		t.Fatalf("search result leaked or missed citation: content=%s citations=%#v", result.Content, result.Citations)
	}
}

func TestPublicToolSearchDoesNotLetHiddenHitsConsumeThePublicLimit(t *testing.T) {
	registry := newPublicToolRegistryForTest(port.Article{Id: 1, Status: port.PublicArticleStatus}, []port.ArticleSearchHit{
		{ArticleSearch: port.ArticleSearch{Id: 2, ArticleTitle: "草稿", Status: 2}},
		{ArticleSearch: port.ArticleSearch{Id: 1, ArticleTitle: "公开文章", ArticleContent: "可见内容", Status: 1}},
		{ArticleSearch: port.ArticleSearch{Id: 3, ArticleTitle: "另一篇公开文章", Status: 1}},
	})
	result, err := registry.Execute(context.Background(), port.PublicToolSearchArticles, json.RawMessage(`{"query":"文章","limit":2}`))
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Results []struct {
			ID int `json:"articleId"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(result.Content), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Results) != 2 || payload.Results[0].ID != 1 || payload.Results[1].ID != 3 {
		t.Fatalf("public search results = %#v, want visible IDs [1 3]", payload.Results)
	}
}

func TestPublicToolSearchFailsClosedWhenStructuredFilterIsUnsupported(t *testing.T) {
	registry := newPublicToolRegistryForTest(port.Article{Id: 1, Status: port.PublicArticleStatus}, nil)
	_, err := registry.Execute(context.Background(), port.PublicToolSearchArticles, json.RawMessage(`{"query":"文章","category":"Go"}`))
	if errors.KindOf(err) != errors.KindUnavailable {
		t.Fatalf("unsupported structured filter error kind = %q, want unavailable", errors.KindOf(err))
	}
}

func TestPublicToolSearchUsesCanonicalInternalCitationURL(t *testing.T) {
	registry := newPublicToolRegistryForTest(port.Article{Id: 1, Status: port.PublicArticleStatus}, []port.ArticleSearchHit{
		{ArticleSearch: port.ArticleSearch{
			Id:             1,
			ArticleTitle:   "公开文章",
			ArticleContent: "可见内容",
			Status:         port.PublicArticleStatus,
		}, Source: &port.ArticleSearchSource{URL: "javascript:alert(1)"}},
	})
	result, err := registry.Execute(context.Background(), port.PublicToolSearchArticles, json.RawMessage(`{"query":"文章"}`))
	if err != nil {
		t.Fatalf("search tool error = %v", err)
	}

	var payload struct {
		Results []struct {
			URL string `json:"url"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(result.Content), &payload); err != nil {
		t.Fatalf("decode search result = %v", err)
	}
	if len(payload.Results) != 1 || payload.Results[0].URL != "/articles/1" {
		t.Fatalf("search result URL = %#v, want canonical internal route", payload.Results)
	}
	if len(result.Citations) != 1 || result.Citations[0].URL != "/articles/1" {
		t.Fatalf("citation URL = %#v, want canonical internal route", result.Citations)
	}
}

func TestPublicToolArgumentsRejectUnknownFields(t *testing.T) {
	registry := newPublicToolRegistryForTest(port.Article{Id: 1, Status: port.PublicArticleStatus}, nil)
	_, err := registry.Execute(context.Background(), port.PublicToolReadArticle, json.RawMessage(`{"articleId":1,"sql":"select 1"}`))
	if errors.KindOf(err) != errors.KindValidation {
		t.Fatalf("unknown argument error kind = %q, want validation", errors.KindOf(err))
	}
	if stderrors.Is(err, context.Canceled) {
		t.Fatal("unexpected cancellation")
	}
}
