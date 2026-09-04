// Package agent contains application-owned public Agent policies and tools.
// It depends only on domain ports; provider SDKs, Redis clients, and xorm do
// not cross this boundary.
package agent

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxPublicToolArguments = 8 << 10
	maxPublicSearchResults = 8
	maxPublicTaxonomyItems = 100
	maxPublicArticleRunes  = 12000
)

type PublicToolDeps struct {
	Articles   port.ArticleRepository
	Searcher   port.ArticleSearcher
	Categories port.CategoryRepository
	Tags       port.TagRepository
	Vitals     port.AgentVitalsProvider
}

func (d PublicToolDeps) validate() error {
	if d.Articles == nil {
		return errors.Invalid("agent.public_tools.dependencies", "article repository is required")
	}
	if d.Searcher == nil {
		return errors.Invalid("agent.public_tools.dependencies", "article searcher is required")
	}
	if d.Categories == nil {
		return errors.Invalid("agent.public_tools.dependencies", "category repository is required")
	}
	if d.Tags == nil {
		return errors.Invalid("agent.public_tools.dependencies", "tag repository is required")
	}
	if d.Vitals == nil {
		return errors.Invalid("agent.public_tools.dependencies", "vitals provider is required")
	}
	return nil
}

type PublicToolRegistry struct {
	articles   port.ArticleRepository
	searcher   port.ArticleSearcher
	categories port.CategoryRepository
	tags       port.TagRepository
	vitals     port.AgentVitalsProvider
}

func NewPublicToolRegistry(deps PublicToolDeps) (*PublicToolRegistry, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &PublicToolRegistry{
		articles:   deps.Articles,
		searcher:   deps.Searcher,
		categories: deps.Categories,
		tags:       deps.Tags,
		vitals:     deps.Vitals,
	}, nil
}

func (r *PublicToolRegistry) Definitions() []port.ToolDefinition {
	// Return fresh parameter buffers so a provider adapter cannot mutate the
	// registry's allowlist through a caller-owned slice or RawMessage.
	return []port.ToolDefinition{
		{
			Name:        port.PublicToolSearchArticles,
			Description: "只搜索公开文章，返回文章标题、摘要和可跳转来源。不得用于修改内容。",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","maxLength":512},"limit":{"type":"integer","minimum":1,"maximum":8},"category":{"type":"string","maxLength":128},"tags":{"type":"array","items":{"type":"string","maxLength":128},"maxItems":16},"year":{"type":"string","maxLength":4},"from":{"type":"string","maxLength":64},"to":{"type":"string","maxLength":64}},"required":["query"],"additionalProperties":false}`),
		},
		{
			Name:        port.PublicToolReadArticle,
			Description: "读取一篇已经公开且未删除的文章；私密、草稿和删除内容永远不可读。",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"articleId":{"type":"integer","minimum":1}},"required":["articleId"],"additionalProperties":false}`),
		},
		{
			Name:        port.PublicToolReadTaxonomy,
			Description: "读取公开站点的分类和标签列表，只读。",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
		},
		{
			Name:        port.PublicToolReadVitals,
			Description: "读取博客公开的基础生命体征摘要，只读，不包含访客身份信息。",
			Parameters:  json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
		},
	}
}

func (r *PublicToolRegistry) Execute(ctx context.Context, name string, arguments json.RawMessage) (port.PublicAgentToolResult, error) {
	if r == nil {
		return port.PublicAgentToolResult{}, errors.Unavailable("agent.public_tool.execute", nil)
	}
	name = strings.TrimSpace(name)
	if !isPublicTool(name) {
		return port.PublicAgentToolResult{}, errors.New(errors.KindForbidden, "agent.public_tool.execute", stderrors.New("tool is not in the public allowlist"))
	}
	if len(arguments) > maxPublicToolArguments {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.arguments", "tool arguments are too large")
	}
	switch name {
	case port.PublicToolSearchArticles:
		return r.searchArticles(ctx, arguments)
	case port.PublicToolReadArticle:
		return r.readArticle(ctx, arguments)
	case port.PublicToolReadTaxonomy:
		return r.readTaxonomy(ctx, arguments)
	case port.PublicToolReadVitals:
		return r.readVitals(ctx, arguments)
	default:
		return port.PublicAgentToolResult{}, errors.New(errors.KindForbidden, "agent.public_tool.execute", stderrors.New("tool is not in the public allowlist"))
	}
}

func isPublicTool(name string) bool {
	switch name {
	case port.PublicToolSearchArticles, port.PublicToolReadArticle, port.PublicToolReadTaxonomy, port.PublicToolReadVitals:
		return true
	default:
		return false
	}
}

type searchArticlesArguments struct {
	Query    string   `json:"query"`
	Limit    int      `json:"limit"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Year     string   `json:"year"`
	From     string   `json:"from"`
	To       string   `json:"to"`
}

func (r *PublicToolRegistry) searchArticles(ctx context.Context, raw json.RawMessage) (port.PublicAgentToolResult, error) {
	var args searchArticlesArguments
	if err := decodeToolArguments(raw, &args); err != nil {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.search_articles", err.Error())
	}
	args.Query = strings.TrimSpace(args.Query)
	if args.Query == "" || utf8.RuneCountInString(args.Query) > 512 {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.search_articles", "query is required and must be at most 512 characters")
	}
	if args.Limit == 0 {
		args.Limit = maxPublicSearchResults
	}
	if args.Limit < 1 || args.Limit > maxPublicSearchResults {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.search_articles", "limit is out of range")
	}
	filter, err := port.ParseKnowledgeFilter(port.KnowledgeFilterInput{
		Category: args.Category,
		Tags:     args.Tags,
		Year:     args.Year,
		From:     args.From,
		To:       args.To,
	})
	if err != nil {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.search_articles", err.Error())
	}
	var hits []port.ArticleSearchHit
	if filtered, ok := r.searcher.(port.ArticleFilteredModeSearcher); ok {
		hits, err = filtered.SearchWithModeAndFilter(ctx, args.Query, port.SearchModeKeyword, filter)
	} else if !filter.Empty() {
		// A structured filter must never be silently dropped. The optional
		// extension is available on the production Meilisearch adapter, but a
		// reduced implementation or a miswired bootstrap must fail closed.
		return port.PublicAgentToolResult{}, errors.Unavailable("agent.public_tool.search_articles", nil)
	} else {
		hits, err = r.searcher.Search(ctx, args.Query)
	}
	if err != nil {
		return port.PublicAgentToolResult{}, err
	}
	type result struct {
		ID      int    `json:"articleId"`
		Title   string `json:"title"`
		Excerpt string `json:"excerpt"`
		URL     string `json:"url"`
	}
	items := make([]result, 0, len(hits))
	citations := make([]port.Citation, 0, len(hits))
	for _, hit := range hits {
		if !port.IsPublicArticle(hit.Status, hit.IsDelete) || hit.Id <= 0 {
			continue
		}
		title := strings.TrimSpace(hit.ArticleTitle)
		excerpt := truncateRunes(hit.ArticleContent, 1000)
		// Public Agent citations must point to the canonical blog route. Search
		// providers may carry an article source URL from indexed or imported
		// content; that value is not trusted presentation data and must not be
		// returned directly to the browser.
		url := publicArticleURL(hit.Id)
		items = append(items, result{ID: hit.Id, Title: title, Excerpt: excerpt, URL: url})
		score := 0.0
		if hit.Relevance != nil {
			score = hit.Relevance.Score
		}
		citations = append(citations, port.Citation{ArticleID: hit.Id, DocumentID: strconv.Itoa(hit.Id), Title: title, URL: url, Score: score})
		if len(items) == args.Limit {
			break
		}
	}
	return marshalToolResult(struct {
		Results []result `json:"results"`
	}{Results: items}, citations)
}

type readArticleArguments struct {
	ArticleID int `json:"articleId"`
}

func (r *PublicToolRegistry) readArticle(ctx context.Context, raw json.RawMessage) (port.PublicAgentToolResult, error) {
	var args readArticleArguments
	if err := decodeToolArguments(raw, &args); err != nil {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.read_article", err.Error())
	}
	if args.ArticleID <= 0 {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.read_article", "articleId must be positive")
	}
	article, err := r.articles.GetArticleByID(ctx, args.ArticleID)
	if err != nil {
		return port.PublicAgentToolResult{}, err
	}
	if !port.IsPublicArticle(article.Status, article.IsDelete) {
		return port.PublicAgentToolResult{}, errors.NotFound("agent.public_tool.read_article")
	}
	content := truncateRunes(article.ArticleContent, maxPublicArticleRunes)
	return marshalToolResult(struct {
		ID      int    `json:"articleId"`
		Title   string `json:"title"`
		Content string `json:"content"`
		URL     string `json:"url"`
	}{ID: article.Id, Title: article.ArticleTitle, Content: content, URL: publicArticleURL(article.Id)}, []port.Citation{{
		ArticleID:  article.Id,
		DocumentID: strconv.Itoa(article.Id),
		Title:      article.ArticleTitle,
		URL:        publicArticleURL(article.Id),
	}})
}

func publicArticleURL(articleID int) string {
	return "/articles/" + strconv.Itoa(articleID)
}

func (r *PublicToolRegistry) readTaxonomy(ctx context.Context, raw json.RawMessage) (port.PublicAgentToolResult, error) {
	if err := rejectNonObjectArguments(raw); err != nil {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.read_taxonomy", err.Error())
	}
	categories, err := r.categories.List(ctx)
	if err != nil {
		return port.PublicAgentToolResult{}, err
	}
	tags, err := r.tags.List(ctx)
	if err != nil {
		return port.PublicAgentToolResult{}, err
	}
	if len(categories) > maxPublicTaxonomyItems {
		categories = categories[:maxPublicTaxonomyItems]
	}
	if len(tags) > maxPublicTaxonomyItems {
		tags = tags[:maxPublicTaxonomyItems]
	}
	type category struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	type tag struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	categoryItems := make([]category, 0, len(categories))
	for _, item := range categories {
		categoryItems = append(categoryItems, category{ID: item.Id, Name: item.CategoryName})
	}
	tagItems := make([]tag, 0, len(tags))
	for _, item := range tags {
		if item != nil {
			tagItems = append(tagItems, tag{ID: item.Id, Name: item.TagName})
		}
	}
	return marshalToolResult(struct {
		Categories []category `json:"categories"`
		Tags       []tag      `json:"tags"`
	}{Categories: categoryItems, Tags: tagItems}, nil)
}

func (r *PublicToolRegistry) readVitals(ctx context.Context, raw json.RawMessage) (port.PublicAgentToolResult, error) {
	if err := rejectNonObjectArguments(raw); err != nil {
		return port.PublicAgentToolResult{}, errors.Invalid("agent.public_tool.read_vitals", err.Error())
	}
	vitals, err := r.vitals.Snapshot(ctx)
	if err != nil {
		return port.PublicAgentToolResult{}, err
	}
	return marshalToolResult(vitals, nil)
}

func decodeToolArguments(raw []byte, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return stderrors.New("tool arguments contain multiple JSON values")
		}
		return err
	}
	return nil
}

func rejectNonObjectArguments(raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	if object == nil {
		return stderrors.New("tool arguments must be an object")
	}
	if len(object) != 0 {
		return stderrors.New("this tool does not accept arguments")
	}
	return nil
}

func marshalToolResult(value any, citations []port.Citation) (port.PublicAgentToolResult, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return port.PublicAgentToolResult{}, errors.Wrap(errors.KindInternal, "agent.public_tool.encode", err)
	}
	return port.PublicAgentToolResult{Content: fmt.Sprintf("%s", raw), Citations: citations}, nil
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "…"
}

var _ port.PublicAgentToolRegistry = (*PublicToolRegistry)(nil)
