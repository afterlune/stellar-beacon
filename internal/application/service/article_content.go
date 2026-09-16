package service

import (
	"html"
	"regexp"
	"strings"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/microcosm-cc/bluemonday"
)

var articleCodeClass = regexp.MustCompile(`^language-[a-zA-Z0-9_-]+$`)
var articleTagPattern = regexp.MustCompile(`(?s)<[^>]*>`)
var articleContentElementPattern = regexp.MustCompile(`(?i)<(?:img|hr|table|ul|ol|blockquote|pre|h[1-6])(?:\s|>)`)

var articleHTMLPolicy = func() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements(
		"p", "br", "hr", "h1", "h2", "h3", "h4", "h5", "h6",
		"strong", "b", "em", "i", "u", "s", "del", "span", "div", "sub", "sup", "mark", "blockquote",
		"pre", "code", "ul", "ol", "li", "table", "thead", "tbody",
		"tfoot", "tr", "th", "td", "a", "img", "figure", "figcaption",
	)
	policy.AllowAttrs("href", "title").OnElements("a")
	policy.AllowAttrs("target").Matching(regexp.MustCompile(`^_blank$`)).OnElements("a")
	policy.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")
	policy.AllowAttrs("class").Matching(articleCodeClass).OnElements("pre", "code")
	policy.AllowStyles(
		"color", "background-color", "font-size", "font-family", "font-weight", "font-style",
		"line-height", "text-align", "text-decoration", "text-indent",
	).OnElements("p", "div", "span", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "li", "td", "th")
	policy.AllowURLSchemes("http", "https", "mailto")
	policy.AllowRelativeURLs(true)
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)
	return policy
}()

// sanitizeArticleHTML keeps the HTML produced by the editor useful while
// making it safe to render through the public site's v-html boundary.
func sanitizeArticleHTML(content string) string {
	return strings.TrimSpace(articleHTMLPolicy.Sanitize(content))
}

// articleHTMLHasContent treats the editor's empty paragraph (`<p><br></p>`)
// as empty while still allowing image-only and separator-based articles.
func articleHTMLHasContent(content string) bool {
	sanitized := sanitizeArticleHTML(content)
	plain := html.UnescapeString(articleTagPattern.ReplaceAllString(sanitized, ""))
	return strings.TrimSpace(plain) != "" || articleContentElementPattern.MatchString(sanitized)
}

// sanitizePublicArticle covers both newly stored HTML and legacy Markdown
// that may contain raw HTML. It is applied before the response/cache boundary
// because older records predate the HTML column and were not sanitized on
// write.
func sanitizePublicArticle(article *port.Article) {
	if article == nil {
		return
	}
	article.ArticleContent = sanitizeArticleHTML(article.ArticleContent)
	article.ArticleContentHTML = sanitizeArticleHTML(article.ArticleContentHTML)
	sanitizePublicCard(&article.PreArticleCard)
	sanitizePublicCard(&article.NextArticleCard)
	for index := range article.RelatedArticles {
		sanitizePublicCard(&article.RelatedArticles[index])
	}
}

func sanitizePublicCard(card *port.ArticleCard) {
	if card == nil {
		return
	}
	card.ArticleContent = sanitizeArticleHTML(card.ArticleContent)
	card.ArticleContentHTML = sanitizeArticleHTML(card.ArticleContentHTML)
}
