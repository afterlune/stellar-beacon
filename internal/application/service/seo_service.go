package service

import (
	"context"
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

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

type SeoService interface {
	RenderArticleHTML(*gin.Context)
	RenderAuthorHTML(*gin.Context)
	RenderSitemap(*gin.Context)
	RenderRobots(*gin.Context)
	RenderFeed(*gin.Context)
}

type MySeoService struct {
	articles port.ArticleRepository
	platform port.PlatformRepository
	baseURL  string
}

func NewSeoService(articles port.ArticleRepository, platform port.PlatformRepository) (*MySeoService, error) {
	if articles == nil {
		return nil, missingServiceDependency("seo", "article repository")
	}
	if platform == nil {
		return nil, missingServiceDependency("seo", "platform repository")
	}
	return &MySeoService{articles: articles, platform: platform, baseURL: strings.TrimRight(config.PublicSiteURL, "/")}, nil
}

func (s *MySeoService) RenderArticleHTML(c *gin.Context) {
	id := strings.TrimSpace(c.Param("articleId"))
	articleID, err := parsePositiveID(id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	article, err := s.articles.GetArticleByID(c.Request.Context(), articleID)
	if err != nil || article.Id == 0 || article.Status != 1 {
		c.AbortWithStatus(http.StatusNotFound)
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

func (s *MySeoService) RenderAuthorHTML(c *gin.Context) {
	handle := strings.TrimSpace(c.Param("handle"))
	author, err := s.platform.GetAuthorByHandle(c.Request.Context(), handle, 0)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			c.AbortWithStatus(http.StatusServiceUnavailable)
		}
		return
	}
	if author.Id <= 0 || strings.TrimSpace(author.Handle) == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	name := authorDisplayName(author)
	description := authorDescription(author, name)
	canonical := s.authorURL(author.Handle)
	title := fmt.Sprintf("%s (@%s) · Stellar Beacon", name, author.Handle)
	articles := s.recentAuthorArticles(c.Request.Context(), author.Id)
	mainEntity := map[string]any{
		"@type":         "Person",
		"name":          name,
		"alternateName": "@" + author.Handle,
		"description":   description,
		"url":           canonical,
	}
	if avatar := strings.TrimSpace(author.Avatar); avatar != "" {
		mainEntity["image"] = avatar
	}
	jsonLD, _ := json.Marshal(map[string]any{
		"@context":   "https://schema.org",
		"@type":      "ProfilePage",
		"url":        canonical,
		"mainEntity": mainEntity,
	})

	twitterCard := "summary"
	var ogImageMeta, twitterImageMeta string
	if avatar := strings.TrimSpace(author.Avatar); avatar != "" {
		escapedAvatar := html.EscapeString(avatar)
		ogImageMeta = fmt.Sprintf(`<meta property="og:image" content="%s">`, escapedAvatar)
		twitterImageMeta = fmt.Sprintf(`<meta name="twitter:image" content="%s">`, escapedAvatar)
		twitterCard = "summary_large_image"
	}

	avatarHTML := ""
	if avatar := strings.TrimSpace(author.Avatar); avatar != "" {
		avatarHTML = fmt.Sprintf(`<img src="%s" alt="%s">`, html.EscapeString(avatar), html.EscapeString(name))
	}
	websiteHTML := ""
	if website := strings.TrimSpace(author.Website); website != "" {
		websiteHTML = fmt.Sprintf(`<p><a href="%s">%s</a></p>`, html.EscapeString(website), html.EscapeString(website))
	}
	statsHTML := fmt.Sprintf(`<dl><div><dt>%d</dt><dd>关注者</dd></div><div><dt>%d</dt><dd>公开文章</dd></div><div><dt>%d</dt><dd>公开随想</dd></div><div><dt>%d</dt><dd>公开系列</dd></div><div><dt>%d</dt><dd>公开书单</dd></div></dl>`, author.FollowerCount, author.ArticleCount, author.TalkCount, author.SeriesCount, author.CollectionCount)
	body := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<meta name="description" content="%s"><link rel="canonical" href="%s">
<meta property="og:type" content="profile"><meta property="og:title" content="%s"><meta property="og:description" content="%s"><meta property="og:url" content="%s">%s
<meta name="twitter:card" content="%s"><meta name="twitter:title" content="%s"><meta name="twitter:description" content="%s">%s
<script type="application/ld+json">%s</script>
<style>body{margin:0;background:#080a12;color:#eef1ff;font-family:system-ui,sans-serif}main{max-width:760px;margin:0 auto;padding:48px 24px}header{display:grid;gap:10px}img{width:112px;height:112px;border-radius:50%%;object-fit:cover}h1{margin:0;font-size:38px}p,dd{color:#aeb6d2;line-height:1.7}a{color:#a9bcff}dl{display:flex;gap:26px;flex-wrap:wrap;margin:24px 0 36px}dt{font-size:24px;font-weight:800}dd{margin:3px 0 0;font-size:12px}ul{display:grid;gap:12px;padding:0;list-style:none}li{padding:14px 0;border-bottom:1px solid #272b3d}li a{font-weight:700;text-decoration:none}</style>
</head><body><main><header>%s<h1>%s</h1><p>@%s</p><p>%s</p>%s</header>%s<section><h2>代表作</h2>%s</section></main>
<noscript>这是 Stellar Beacon 作者公开主页；启用 JavaScript 可查看完整交互页面。</noscript>
</body></html>`, html.EscapeString(title), html.EscapeString(description), html.EscapeString(canonical), html.EscapeString(title), html.EscapeString(description), html.EscapeString(canonical), ogImageMeta, twitterCard, html.EscapeString(title), html.EscapeString(description), twitterImageMeta, string(jsonLD), avatarHTML, html.EscapeString(name), html.EscapeString(author.Handle), html.EscapeString(author.Intro), websiteHTML, statsHTML, renderAuthorArticlesHTML(s, articles))
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(body))
}

func (s *MySeoService) RenderSitemap(c *gin.Context) {
	articles, _, err := s.articles.ListArchives(c.Request.Context(), 1, 10000)
	if err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	type location struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod,omitempty"`
	}
	entries := []location{{Loc: s.baseURL + "/"}, {Loc: s.baseURL + "/about"}, {Loc: s.baseURL + "/archives"}, {Loc: s.baseURL + "/authors"}}
	for _, article := range articles {
		entries = append(entries, location{Loc: s.articleURL(article.Id), LastMod: article.CreateTime.Format("2006-01-02")})
	}
	authorCount := 0
	for current := 1; current <= 100 && authorCount < 10000; current++ {
		authors, total, err := s.platform.ListAuthors(c.Request.Context(), current, 100, 0, port.AuthorSortActive)
		if err != nil {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		for _, author := range authors {
			if author.Id <= 0 || strings.TrimSpace(author.Handle) == "" {
				continue
			}
			entry := location{Loc: s.authorURL(author.Handle)}
			if author.LastPublishedAt != nil {
				entry.LastMod = author.LastPublishedAt.Format("2006-01-02")
			}
			entries = append(entries, entry)
			authorCount++
			if authorCount >= 10000 {
				break
			}
		}
		if len(authors) == 0 || authorCount >= total {
			break
		}
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
		c.AbortWithStatus(http.StatusServiceUnavailable)
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

func (s *MySeoService) authorURL(handle string) string {
	return s.baseURL + "/u/" + url.PathEscape(strings.TrimSpace(handle))
}

func (s *MySeoService) recentAuthorArticles(ctx context.Context, authorID int) []*port.ArticleCard {
	articles, _, err := s.platform.ListAuthorArticlesHot(ctx, authorID, 1, 6)
	if err != nil || len(articles) == 0 {
		articles, _, err = s.platform.ListAuthorArticles(ctx, authorID, 1, 6)
	}
	if err != nil {
		return nil
	}
	if len(articles) > 6 {
		return articles[:6]
	}
	return articles
}

func authorDisplayName(author port.AuthorCard) string {
	if name := strings.TrimSpace(author.Nickname); name != "" {
		return name
	}
	return strings.TrimSpace(author.Handle)
}

func authorDescription(author port.AuthorCard, name string) string {
	if intro := strings.TrimSpace(author.Intro); intro != "" {
		return truncateRunes(intro, 180)
	}
	return fmt.Sprintf("%s 的公开主页，收录 %d 篇文章、%d 条随想和 %d 个系列。", name, author.ArticleCount, author.TalkCount, author.SeriesCount)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}

func renderAuthorArticlesHTML(s *MySeoService, articles []*port.ArticleCard) string {
	if len(articles) == 0 {
		return "<p>这位作者还没有公开文章。</p>"
	}
	var builder strings.Builder
	builder.WriteString("<ul>")
	for _, article := range articles {
		if article == nil || article.Id <= 0 || strings.TrimSpace(article.ArticleTitle) == "" {
			continue
		}
		fmt.Fprintf(&builder, `<li><a href="%s">%s</a> <span>%s</span></li>`, html.EscapeString(s.articleURL(article.Id)), html.EscapeString(article.ArticleTitle), html.EscapeString(article.CreateTime.Format("2006-01-02")))
	}
	builder.WriteString("</ul>")
	return builder.String()
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
