package ormInit

import (
	"testing"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"xorm.io/xorm"
)

// xorm's snake mapper splits acronyms, so a field like `ArticleContentHTML` maps
// to `article_content_h_t_m_l`.  Columns whose name is not derivable from the Go
// field name must therefore pin it in the xorm tag; otherwise every statement
// touching that column fails against the real schema while the unit tests, which
// never build SQL, stay green.
func TestArticleRichTextColumnMapping(t *testing.T) {
	engine, err := xorm.NewEngine("postgres", "postgres://unused:unused@127.0.0.1:1/unused?sslmode=disable")
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	defer func() { _ = engine.Close() }()

	table, err := engine.TableInfo(&entity.TArticle{})
	if err != nil {
		t.Fatalf("parse t_article mapping: %v", err)
	}
	if table.GetColumn("article_content_html") == nil {
		t.Fatalf("t_article has no article_content_html column mapping; columns: %v", table.ColumnsSeq())
	}
}
