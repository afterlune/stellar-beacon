package service

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

type SeoService interface {
	RenderArticleHTML(*gin.Context)
	RenderSitemap(*gin.Context)
	RenderRobots(*gin.Context)
	RenderFeed(*gin.Context)
}

type MySeoService struct {
	articles port.ArticleRepository
	baseURL  string
}

func NewSeoService(articles port.ArticleRepository) (*MySeoService, error) {
	if articles == nil {
		return nil, missingServiceDependency("seo", "article repository")
	}
	return &MySeoService{articles: articles, baseURL: strings.TrimRight(config.PublicSiteURL, "/")}, nil
}

func (s *MySeoService) RenderArticleHTML(c *gin.Context) {
	id := strings.TrimSpace(c.Param("articleId"))
	articleID, err := parsePositiveID(id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	article, err := s.articles.GetArticleByID(c.Request.Context(), articleID)
	if err != nil || article.Id == 0 || article.Status != 1 {
		c.Status(http.StatusNotFound)
		return
	}
	canonical := s.articleURL(article.Id)
	description := articleDescription(article.ArticleContent)
	cover := strings.TrimSpace(article.ArticleCover)
	jsonLD, _ := json.Marshal(map[string]any{
		"@context":      "https://schema.org",
		"@type":         "Article",
		"headline":      article.ArticleTitle,
		"description":   description,
		"datePublished": article.CreateTime.Format(time.RFC3339),
		"dateModified":  article.UpdateTime.Format(time.RFC3339),
		"mainEntityOfPage": map[string]string{
			"@type": "WebPage",
			"@id":   canonical,
		},
	})

	var imageMeta string
	if cover != "" {
		imageMeta = fmt.Sprintf(`<meta property="og:image" content="%s"><meta name="twitter:image" content="%s">`, html.EscapeString(cover), html.EscapeString(cover))
	}
	body := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s · Stellar Beacon</title>
<meta name="description" content="%s"><link rel="canonical" href="%s">
<meta property="og:type" content="article"><meta property="og:title" content="%s"><meta property="og:description" content="%s"><meta property="og:url" content="%s">%s
<meta name="twitter:card" content="summary_large_image"><meta name="twitter:title" content="%s"><meta name="twitter:description" content="%s">
<script type="application/ld+json">%s</script>
</head><body><main><article><p class="eyebrow">STELLAR BEACON / ARTICLE</p><h1>%s</h1><p class="meta">%s</p><div class="content">%s</div></article></main>
<noscript>本文由 Stellar Beacon 输出；启用 JavaScript 可查看完整交互页面。</noscript>
</body></html>`,
		html.EscapeString(article.ArticleTitle), html.EscapeString(description), html.EscapeString(canonical),
		html.EscapeString(article.ArticleTitle), html.EscapeString(description), html.EscapeString(canonical), imageMeta,
		html.EscapeString(article.ArticleTitle), html.EscapeString(description), string(jsonLD),
		html.EscapeString(article.ArticleTitle), html.EscapeString(article.CreateTime.Format("2006-01-02")), renderPlainMarkdown(article.ArticleContent))
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(body))
}

func (s *MySeoService) RenderSitemap(c *gin.Context) {
	articles, _, err := s.articles.ListArchives(c.Request.Context(), 1, 10000)
	if err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}
	type location struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod,omitempty"`
	}
	entries := []location{{Loc: s.baseURL + "/"}, {Loc: s.baseURL + "/about"}, {Loc: s.baseURL + "/archives"}}
	for _, article := range articles {
		entries = append(entries, location{Loc: s.articleURL(article.Id), LastMod: article.CreateTime.Format("2006-01-02")})
	}
	data, _ := xmlMarshal(struct {
		XMLName struct{}   `xml:"urlset"`
		XMLNS   string     `xml:"xmlns,attr"`
		URL     []location `xml:"url"`
	}{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9", URL: entries})
	c.Data(http.StatusOK, "application/xml; charset=utf-8", data)
}

func (s *MySeoService) RenderRobots(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("User-agent: *\nAllow: /\nDisallow: /api/\nSitemap: "+s.baseURL+"/sitemap.xml\n"))
}

func (s *MySeoService) RenderFeed(c *gin.Context) {
	articles, _, err := s.articles.ListArchives(c.Request.Context(), 1, 50)
	if err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}
	type item struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
		GUID  string `xml:"guid"`
		Date  string `xml:"pubDate"`
		Desc  string `xml:"description"`
	}
	items := make([]item, 0, len(articles))
	for _, article := range articles {
		items = append(items, item{Title: article.ArticleTitle, Link: s.articleURL(article.Id), GUID: s.articleURL(article.Id), Date: article.CreateTime.Format(time.RFC1123Z), Desc: articleDescription(article.ArticleContent)})
	}
	data, _ := xmlMarshal(struct {
		XMLName struct{} `xml:"rss"`
		Version string   `xml:"version,attr"`
		Channel struct {
			Title string `xml:"title"`
			Link  string `xml:"link"`
			Items []item `xml:"item"`
		} `xml:"channel"`
	}{Version: "2.0", Channel: struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
		Items []item `xml:"item"`
	}{Title: "Stellar Beacon", Link: s.baseURL, Items: items}})
	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", data)
}

func (s *MySeoService) articleURL(id int) string {
	return s.baseURL + "/articles/" + url.PathEscape(fmt.Sprint(id))
}

func parsePositiveID(value string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid article id")
	}
	return id, nil
}

var whitespace = regexp.MustCompile(`\s+`)

func articleDescription(value string) string {
	plain := strings.TrimSpace(whitespace.ReplaceAllString(value, " "))
	plain = strings.TrimLeft(plain, "#>-* ")
	if len(plain) > 180 {
		plain = plain[:180] + "…"
	}
	return plain
}

func renderPlainMarkdown(value string) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	paragraphs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		level := 0
		for level < len(line) && level < 3 && line[level] == '#' {
			level++
		}
		if level > 0 && level < len(line) && line[level] == ' ' {
			paragraphs = append(paragraphs, fmt.Sprintf("<h%d>%s</h%d>", min(level+1, 4), html.EscapeString(strings.TrimSpace(line[level:])), min(level+1, 4)))
			continue
		}
		paragraphs = append(paragraphs, "<p>"+html.EscapeString(line)+"</p>")
	}
	return strings.Join(paragraphs, "\n")
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// xmlMarshal keeps XML encoding in one place and makes the response helpers
// easy to test without exposing encoding/xml from the service API.
func xmlMarshal(value any) ([]byte, error) {
	return xml.Marshal(value)
}

var _ SeoService = (*MySeoService)(nil)
