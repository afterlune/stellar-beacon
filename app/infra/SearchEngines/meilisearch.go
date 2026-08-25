package SearchEngines

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"log/slog"
	"time"

	"github.com/goccy/go-json"
	"github.com/meilisearch/meilisearch-go"
)

func GetClient() meilisearch.ServiceManager {
	meili := new(config.MeiliSearch).MeiliSearch()
	return meilisearch.New(meili.URL, meilisearch.WithAPIKey(meili.ApiKey))
}

func Search(keywords string) []interface{} {
	client := GetClient()
	searchResponse, err := client.Index("articles").Search(keywords, &meilisearch.SearchRequest{
		AttributesToRetrieve:  []string{"*"},
		Limit:                 1000,
		Offset:                0,
		AttributesToHighlight: []string{"articleTitle", "articleContent"},
		CropLength:            50,
		HighlightPreTag:       shared.PRE_TAG,
		HighlightPostTag:      shared.POST_TAG,
	})
	if err != nil {
		slog.Error("search articles failed", "error", err)
	}
	if searchResponse.EstimatedTotalHits == 0 {
		return []interface{}{}
	}
	hits := make([]interface{}, len(searchResponse.Hits))
	for index, hit := range searchResponse.Hits {
		hits[index] = hit
	}
	return hits
}

func docSyncTask() {
	for {
		client := GetClient()
		var articleSearchDTOs []model.ArticleSearchDTO
		engine := ormInit.GetEngine()
		if engine == nil {
			time.Sleep(time.Minute)
			continue
		}
		err := engine.SQL("select id, article_title, SUBSTR(article_content, 1, 500) AS " +
			"article_content, is_delete, status from t_article where is_delete = 0 and status = 1").Find(&articleSearchDTOs)
		if err != nil {
			slog.Error("load articles for search synchronization failed", "error", err)
			continue
		}
		var docs []map[string]interface{}
		data, _ := json.Marshal(articleSearchDTOs)
		err = json.Unmarshal(data, &docs)
		if err != nil {
			slog.Error("marshal search documents failed", "error", err)
			continue
		}
		_, err = client.Index("articles").UpdateDocuments(docs, nil)
		if err != nil {
			slog.Error("update search documents failed", "error", err)
			continue
		}
		slog.Info("search documents synchronized")
		time.Sleep(time.Minute * 10)
	}
}

func init() {
	if err := config.Validate(); err != nil {
		slog.Warn("search initialization skipped", "error", err)
		return
	}
	client := GetClient()
	index, err1 := client.GetIndex("articles")
	if index == nil && err1 != nil {
		engine := ormInit.GetEngine()
		if engine == nil {
			slog.Warn("search index initialization skipped: database is unavailable")
			return
		}
		var articleSearchDTOs []model.ArticleSearchDTO
		err := engine.SQL("select id, article_title, SUBSTR(article_content, 1, 500) AS " +
			"article_content, is_delete, status from t_article where is_delete = 0 and status = 1").Find(&articleSearchDTOs)
		if err != nil {
			slog.Error("load initial search documents failed", "error", err)
		}
		var docs []map[string]interface{}
		data, _ := json.Marshal(articleSearchDTOs)
		err = json.Unmarshal(data, &docs)
		if err != nil {
			slog.Error("marshal initial search documents failed", "error", err)
		}
		_, err = client.Index("articles").AddDocuments(docs, nil)
		if err != nil {
			slog.Error("create search index documents failed", "error", err)
		}
	}
	// 开启一个协程执行search同步工作
	go docSyncTask()
}
