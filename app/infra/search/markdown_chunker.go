package search

import (
	"html"
	"regexp"
	"strings"
	"unicode"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const (
	DefaultMarkdownChunkSize    = 500
	DefaultMarkdownChunkOverlap = 80
)

type MarkdownChunkerConfig struct {
	ChunkSize   int
	OverlapSize int
}

type MarkdownChunker struct {
	config MarkdownChunkerConfig
}

var (
	markdownImagePattern     = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	markdownLinkPattern      = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	markdownReferencePattern = regexp.MustCompile(`\[([^\]]+)\]\[[^\]]*\]`)
	inlineCodePattern        = regexp.MustCompile("`([^`\\n]+)`")
	htmlAutolinkPattern      = regexp.MustCompile(`<((?:https?://|mailto:)[^>]+)>`)
	htmlCommentPattern       = regexp.MustCompile(`<!--.*?-->`)
	htmlTagPattern           = regexp.MustCompile(`</?[A-Za-z][^>]*>`)
	escapedMarkdownPattern   = regexp.MustCompile(`\\([\\` + "`" + `*_{}\[\]()#+.!|>\-])`)
	strongPattern            = regexp.MustCompile(`\*\*([^*\n]+)\*\*|__([^_\n]+)__`)
	strikePattern            = regexp.MustCompile(`~~([^~\n]+)~~`)
	emphasisPattern          = regexp.MustCompile(`\*([^*\n]+)\*|_([^_\n]+)_`)
	headingPattern           = regexp.MustCompile(`^#{1,6}\s+`)
	unorderedListPattern     = regexp.MustCompile(`^[-+*]\s+`)
	orderedListPattern       = regexp.MustCompile(`^\d+[.)]\s+`)
	taskPattern              = regexp.MustCompile(`^\[[ xX]\]\s+`)
	horizontalRulePattern    = regexp.MustCompile(`^(?:-{3,}|_{3,}|\*{3,})$`)
	whitespacePattern        = regexp.MustCompile(`[ \t]+`)
)

func DefaultMarkdownChunkerConfig() MarkdownChunkerConfig {
	return MarkdownChunkerConfig{
		ChunkSize:   DefaultMarkdownChunkSize,
		OverlapSize: DefaultMarkdownChunkOverlap,
	}
}

func NewMarkdownChunker(config MarkdownChunkerConfig) (MarkdownChunker, error) {
	if config == (MarkdownChunkerConfig{}) {
		config = DefaultMarkdownChunkerConfig()
	} else if config.ChunkSize == 0 {
		config.ChunkSize = DefaultMarkdownChunkSize
	}
	if config.ChunkSize <= 0 {
		return MarkdownChunker{}, apperrors.Invalid("search.markdown_chunker.chunk_size", "chunk size must be positive")
	}
	if config.OverlapSize < 0 || config.OverlapSize >= config.ChunkSize {
		return MarkdownChunker{}, apperrors.Invalid("search.markdown_chunker.overlap", "overlap must be non-negative and smaller than chunk size")
	}
	return MarkdownChunker{config: config}, nil
}

func (c MarkdownChunker) Chunk(markdown string) []string {
	config := c.config
	if config == (MarkdownChunkerConfig{}) {
		config = DefaultMarkdownChunkerConfig()
	}
	if config.ChunkSize <= 0 || config.OverlapSize < 0 || config.OverlapSize >= config.ChunkSize {
		return nil
	}
	cleaned := CleanMarkdown(markdown)
	if cleaned == "" {
		return nil
	}
	runes := []rune(cleaned)
	chunks := make([]string, 0, (len(runes)+config.ChunkSize-config.OverlapSize-1)/(config.ChunkSize-config.OverlapSize))
	for start := 0; start < len(runes); {
		for start < len(runes) && unicode.IsSpace(runes[start]) {
			start++
		}
		if start >= len(runes) {
			break
		}
		end := start + config.ChunkSize
		if end > len(runes) {
			end = len(runes)
		} else {
			end = preferChunkBoundary(runes, start, end, config.ChunkSize)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end >= len(runes) {
			break
		}
		next := end - config.OverlapSize
		if next <= start {
			next = start + 1
		}
		start = next
	}
	return chunks
}

func ChunkMarkdown(markdown string, config MarkdownChunkerConfig) ([]string, error) {
	chunker, err := NewMarkdownChunker(config)
	if err != nil {
		return nil, err
	}
	return chunker.Chunk(markdown), nil
}

// CleanMarkdown removes presentation syntax while retaining the searchable
// text. Fenced and indented code are handled before normal Markdown cleanup so
// punctuation, indentation and comments inside code cannot be rewritten.
func CleanMarkdown(markdown string) string {
	if markdown == "" {
		return ""
	}
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")
	markdown = strings.ReplaceAll(markdown, "\r", "\n")
	lines := strings.Split(markdown, "\n")
	blocks := make([]string, 0, len(lines)/3+1)
	textLines := make([]string, 0, len(lines))
	codeLines := make([]string, 0, len(lines))
	inFence := false
	fenceChar := byte(0)
	fenceLength := 0
	fenceLanguage := ""
	inIndentedCode := false

	flushText := func() {
		if len(textLines) == 0 {
			return
		}
		cleaned := make([]string, 0, len(textLines))
		for _, line := range textLines {
			line = cleanMarkdownLine(line)
			if line != "" {
				cleaned = append(cleaned, line)
			}
		}
		if len(cleaned) > 0 {
			blocks = append(blocks, strings.Join(cleaned, "\n"))
		}
		textLines = textLines[:0]
	}
	flushCode := func() {
		if len(codeLines) == 0 {
			return
		}
		code := strings.TrimRight(strings.Join(codeLines, "\n"), "\n")
		if code != "" {
			marker := "[code]"
			if fenceLanguage != "" {
				marker = "[code:" + fenceLanguage + "]"
			}
			blocks = append(blocks, marker+"\n"+code+"\n[/code]")
		}
		codeLines = codeLines[:0]
		fenceLanguage = ""
	}

	for _, line := range lines {
		if inFence {
			if isFenceClose(line, fenceChar, fenceLength) {
				flushCode()
				inFence = false
				fenceChar = 0
				fenceLength = 0
				continue
			}
			codeLines = append(codeLines, line)
			continue
		}

		if inIndentedCode {
			if isIndentedCodeLine(line) {
				codeLines = append(codeLines, removeCodeIndent(line))
				continue
			}
			if strings.TrimSpace(line) == "" {
				codeLines = append(codeLines, "")
				continue
			}
			flushCode()
			inIndentedCode = false
		}

		if char, length, language, ok := parseFence(line); ok {
			flushText()
			inFence = true
			fenceChar = char
			fenceLength = length
			fenceLanguage = language
			continue
		}
		if isIndentedCodeLine(line) {
			flushText()
			inIndentedCode = true
			codeLines = append(codeLines, removeCodeIndent(line))
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushText()
			continue
		}
		textLines = append(textLines, line)
	}

	if inFence || inIndentedCode {
		flushCode()
	} else {
		flushText()
	}
	return strings.TrimSpace(strings.Join(blocks, "\n\n"))
}

func cleanMarkdownLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if horizontalRulePattern.MatchString(line) {
		return ""
	}
	line = html.UnescapeString(line)
	line = htmlCommentPattern.ReplaceAllString(line, " ")
	line = strings.ReplaceAll(line, "<br>", " ")
	line = strings.ReplaceAll(line, "<br/>", " ")
	line = strings.ReplaceAll(line, "<br />", " ")
	line = markdownImagePattern.ReplaceAllString(line, "$1")
	line = markdownLinkPattern.ReplaceAllString(line, "$1")
	line = markdownReferencePattern.ReplaceAllString(line, "$1")
	line = inlineCodePattern.ReplaceAllString(line, "$1")
	line = htmlAutolinkPattern.ReplaceAllString(line, "$1")
	line = htmlTagPattern.ReplaceAllString(line, " ")
	line = escapedMarkdownPattern.ReplaceAllString(line, "$1")
	line = headingPattern.ReplaceAllString(line, "")
	line = taskPattern.ReplaceAllString(line, "")
	line = unorderedListPattern.ReplaceAllString(line, "")
	line = orderedListPattern.ReplaceAllString(line, "")
	line = replaceDelimitedText(strongPattern, line)
	line = replaceDelimitedText(strikePattern, line)
	line = replaceDelimitedText(emphasisPattern, line)
	line = strings.ReplaceAll(line, "|", " ")
	line = whitespacePattern.ReplaceAllString(line, " ")
	return strings.TrimSpace(line)
}

func replaceDelimitedText(pattern *regexp.Regexp, value string) string {
	return pattern.ReplaceAllStringFunc(value, func(match string) string {
		groups := pattern.FindStringSubmatch(match)
		for _, group := range groups[1:] {
			if group != "" {
				return group
			}
		}
		return match
	})
}

func parseFence(line string) (byte, int, string, bool) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 || (trimmed[0] != '`' && trimmed[0] != '~') {
		return 0, 0, "", false
	}
	char := trimmed[0]
	length := 0
	for length < len(trimmed) && trimmed[length] == char {
		length++
	}
	if length < 3 {
		return 0, 0, "", false
	}
	if char == '`' && strings.Contains(trimmed[length:], "`") {
		return 0, 0, "", false
	}
	language := ""
	if fields := strings.Fields(trimmed[length:]); len(fields) > 0 {
		language = fields[0]
	}
	return char, length, language, true
}

func isFenceClose(line string, char byte, minimumLength int) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < minimumLength || len(trimmed) == 0 || trimmed[0] != char {
		return false
	}
	length := 0
	for length < len(trimmed) && trimmed[length] == char {
		length++
	}
	return length >= minimumLength && strings.TrimSpace(trimmed[length:]) == ""
}

func isIndentedCodeLine(line string) bool {
	return strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ")
}

func removeCodeIndent(line string) string {
	if strings.HasPrefix(line, "\t") {
		return line[1:]
	}
	return strings.TrimPrefix(line, "    ")
}

func preferChunkBoundary(runes []rune, start, end, chunkSize int) int {
	minimum := start + chunkSize/2
	for index := end - 1; index > minimum; index-- {
		if runes[index] == '\n' || unicode.IsSpace(runes[index]) {
			return index
		}
	}
	return end
}

// BuildArticleChunkDocuments connects the schema from M2-03 to this
// chunker. It performs no database or Meilisearch writes, which keeps retries
// and later durable jobs responsible for persistence.
func BuildArticleChunkDocuments(article port.Article, tags []string, chunker MarkdownChunker) ([]port.ArticleChunkDocument, error) {
	if article.Id <= 0 {
		return nil, apperrors.Invalid("search.article_chunks.build", "article id must be positive")
	}
	if !port.IsPublicArticle(article.Status, article.IsDelete) {
		return nil, apperrors.Invalid("search.article_chunks.build", "only public articles may be indexed")
	}
	chunks := chunker.Chunk(article.ArticleContent)
	if len(chunks) == 0 {
		return nil, apperrors.Invalid("search.article_chunks.build", "article content produced no searchable chunks")
	}
	publishedAt := article.CreateTime
	if publishedAt.IsZero() {
		publishedAt = article.UpdateTime
	}
	documents := make([]port.ArticleChunkDocument, 0, len(chunks))
	for index, text := range chunks {
		id, err := ArticleChunkID(article.Id, index)
		if err != nil {
			return nil, err
		}
		document, err := NormalizeArticleChunkDocument(port.ArticleChunkDocument{
			ID:           id,
			ArticleID:    article.Id,
			ArticleTitle: article.ArticleTitle,
			Text:         text,
			Category:     article.CategoryName,
			Tags:         append([]string(nil), tags...),
			ArticleURL:   article.OriginalUrl,
			PublishedAt:  publishedAt,
			ChunkIndex:   index,
			Status:       article.Status,
			IsDelete:     article.IsDelete,
		})
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, nil
}
