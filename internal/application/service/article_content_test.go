package service

import (
	"strings"
	"testing"
)

func TestSanitizeArticleHTMLKeepsFormattingAndRemovesExecutableMarkup(t *testing.T) {
	input := `<h2 style="text-align:center">Signal</h2><p><strong>hello</strong> <a href="https://example.com" onclick="alert(1)">read</a></p><img src="https://example.com/cover.jpg" onerror="alert(2)" /><script>alert(3)</script>`
	got := sanitizeArticleHTML(input)

	for _, want := range []string{"<h2 style=\"text-align: center\">Signal</h2>", "<strong>hello</strong>", `href="https://example.com"`, `src="https://example.com/cover.jpg"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("sanitized article lost %q: %s", want, got)
		}
	}
	for _, forbidden := range []string{"<script", "onclick", "onerror", "alert("} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("sanitized article still contains %q: %s", forbidden, got)
		}
	}
}

func TestSanitizeArticleHTMLRemovesDangerousURLs(t *testing.T) {
	got := sanitizeArticleHTML(`<a href="javascript:alert(1)">bad</a><img src="data:text/html;base64,deadbeef" />`)
	if strings.Contains(strings.ToLower(got), "javascript:") || strings.Contains(strings.ToLower(got), "data:") {
		t.Fatalf("dangerous URL survived sanitization: %s", got)
	}
}

func TestArticleHTMLHasContentRejectsEmptyEditorParagraph(t *testing.T) {
	if articleHTMLHasContent("<p><br></p><p>\n</p>") {
		t.Fatal("expected an empty editor paragraph to be rejected")
	}
	if !articleHTMLHasContent(`<p>text</p>`) || !articleHTMLHasContent(`<p><img src="/image.png"></p>`) {
		t.Fatal("expected text and image content to be accepted")
	}
}
