package search

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestCleanMarkdownRemovesPresentationSyntax(t *testing.T) {
	markdown := "# 标题\n\n这是 **重要** 的 `code`，请访问 [主页](https://example.test)。\n![封面](cover.png)\n\n<p>HTML 文本</p>"

	cleaned := CleanMarkdown(markdown)
	for _, want := range []string{"标题", "重要", "code", "主页", "封面", "HTML 文本"} {
		if !strings.Contains(cleaned, want) {
			t.Fatalf("CleanMarkdown() = %q, missing %q", cleaned, want)
		}
	}
	for _, unwanted := range []string{"# ", "**", "[主页]", "![", "<p>", "</p>"} {
		if strings.Contains(cleaned, unwanted) {
			t.Fatalf("CleanMarkdown() = %q, still contains presentation syntax %q", cleaned, unwanted)
		}
	}
}

func TestCleanMarkdownPreservesFencedAndIndentedCode(t *testing.T) {
	markdown := "```go\n# not a heading\n**not bold**\nfmt.Println(\"你好\")\n```\n\n    if a_b < 2 {\n        return\n    }\n\n正文"

	cleaned := CleanMarkdown(markdown)
	if !strings.Contains(cleaned, "[code:go]\n# not a heading\n**not bold**\nfmt.Println(\"你好\")\n[/code]") {
		t.Fatalf("fenced code was not preserved: %q", cleaned)
	}
	if !strings.Contains(cleaned, "[code]\nif a_b < 2 {\n    return\n}\n[/code]") {
		t.Fatalf("indented code was not preserved: %q", cleaned)
	}
	if !strings.HasSuffix(cleaned, "正文") {
		t.Fatalf("text after indented code was lost: %q", cleaned)
	}
}

func TestMarkdownChunkerUsesRuneLengthAndOverlap(t *testing.T) {
	chunker, err := NewMarkdownChunker(MarkdownChunkerConfig{ChunkSize: 500, OverlapSize: 100})
	if err != nil {
		t.Fatalf("NewMarkdownChunker() error = %v", err)
	}

	chunks := chunker.Chunk(strings.Repeat("甲", 1200))
	if len(chunks) != 3 {
		t.Fatalf("chunk count = %d, want 3", len(chunks))
	}
	for index, chunk := range chunks {
		if got := len([]rune(chunk)); got > 500 {
			t.Fatalf("chunk %d rune length = %d, want <= 500", index, got)
		}
		if strings.ContainsRune(chunk, '\uFFFD') {
			t.Fatalf("chunk %d contains a replacement rune: %q", index, chunk)
		}
	}
	if got := len([]rune(chunks[0])); got != 500 {
		t.Fatalf("first chunk rune length = %d, want 500", got)
	}
	if got := len([]rune(chunks[1])); got != 500 {
		t.Fatalf("second chunk rune length = %d, want 500", got)
	}
	if !strings.HasPrefix(chunks[1], string([]rune(chunks[0])[400:])) {
		t.Fatal("second chunk does not contain the configured overlap from the first chunk")
	}
	if !strings.HasPrefix(chunks[2], string([]rune(chunks[1])[400:])) {
		t.Fatal("third chunk does not contain the configured overlap from the second chunk")
	}
}

func TestMarkdownChunkerPrefersWhitespaceBoundary(t *testing.T) {
	chunker, err := NewMarkdownChunker(MarkdownChunkerConfig{ChunkSize: 24, OverlapSize: 4})
	if err != nil {
		t.Fatalf("NewMarkdownChunker() error = %v", err)
	}

	chunks := chunker.Chunk("alpha bravo charlie delta echo foxtrot golf hotel")
	if len(chunks) < 2 {
		t.Fatalf("chunk count = %d, want at least 2", len(chunks))
	}
	if chunks[0] != "alpha bravo charlie" {
		t.Fatalf("chunk boundary did not prefer a complete word: %q", chunks[0])
	}
}

func TestNewMarkdownChunkerValidatesConfiguration(t *testing.T) {
	defaults, err := NewMarkdownChunker(MarkdownChunkerConfig{})
	if err != nil {
		t.Fatalf("default configuration error = %v", err)
	}
	if got := len(defaults.Chunk(strings.Repeat("x", DefaultMarkdownChunkSize))); got != 1 {
		t.Fatalf("default chunker returned %d chunks, want 1", got)
	}
	noOverlap, err := NewMarkdownChunker(MarkdownChunkerConfig{ChunkSize: 10, OverlapSize: 0})
	if err != nil {
		t.Fatalf("zero-overlap configuration error = %v", err)
	}
	if got := noOverlap.Chunk(strings.Repeat("x", 20)); len(got) != 2 {
		t.Fatalf("zero-overlap chunker returned %d chunks, want 2", len(got))
	}

	for _, config := range []MarkdownChunkerConfig{
		{ChunkSize: -1, OverlapSize: 1},
		{ChunkSize: 10, OverlapSize: -1},
		{ChunkSize: 10, OverlapSize: 10},
	} {
		if _, err := NewMarkdownChunker(config); err == nil {
			t.Fatalf("NewMarkdownChunker(%+v) error = nil", config)
		}
	}
}

func TestBuildArticleChunkDocumentsAddsStableMetadata(t *testing.T) {
	chunker, err := NewMarkdownChunker(MarkdownChunkerConfig{ChunkSize: 20, OverlapSize: 5})
	if err != nil {
		t.Fatalf("NewMarkdownChunker() error = %v", err)
	}
	publishedAt := time.Date(2026, 8, 28, 12, 34, 56, 0, time.FixedZone("CST", 8*60*60))
	article := port.Article{
		Id:             42,
		ArticleTitle:   "  Agent 入门  ",
		ArticleContent: "# 正文\n\n这是用于索引的文章内容。",
		CategoryName:   "  Engineering ",
		OriginalUrl:    " /articles/42 ",
		Status:         port.PublicArticleStatus,
		CreateTime:     publishedAt,
	}

	documents, err := BuildArticleChunkDocuments(article, []string{" Go ", "Go", "agent"}, chunker)
	if err != nil {
		t.Fatalf("BuildArticleChunkDocuments() error = %v", err)
	}
	if len(documents) == 0 {
		t.Fatal("BuildArticleChunkDocuments() returned no documents")
	}
	first := documents[0]
	if first.ID != "article-42-chunk-0" || first.ArticleID != 42 || first.ArticleTitle != "Agent 入门" {
		t.Fatalf("unexpected document identity or title: %+v", first)
	}
	if first.Category != "Engineering" || first.ArticleURL != "/articles/42" {
		t.Fatalf("unexpected document source metadata: %+v", first)
	}
	if len(first.Tags) != 2 || first.Tags[0] != "Go" || first.Tags[1] != "agent" {
		t.Fatalf("unexpected normalized tags: %v", first.Tags)
	}
	if first.PublishedAt.Location() != time.UTC || first.PublishedAt.UnixNano() != publishedAt.UnixNano() {
		t.Fatalf("unexpected published time: %v", first.PublishedAt)
	}
	for index, document := range documents {
		if document.ChunkIndex != index || document.ID != "article-42-chunk-"+strconv.Itoa(index) || !document.IsPublic() {
			t.Fatalf("document %d has unstable identity or visibility: %+v", index, document)
		}
	}

	article.IsDelete = 1
	if _, err := BuildArticleChunkDocuments(article, nil, chunker); err == nil {
		t.Fatal("deleted article was accepted")
	}
	article.IsDelete = 0
	article.Status = 2
	if _, err := BuildArticleChunkDocuments(article, nil, chunker); err == nil {
		t.Fatal("private article was accepted")
	}
	article.Status = port.PublicArticleStatus
	article.IsDelete = 0
	article.ArticleContent = "   "
	if _, err := BuildArticleChunkDocuments(article, nil, chunker); err == nil {
		t.Fatal("empty article content was accepted")
	}
}
