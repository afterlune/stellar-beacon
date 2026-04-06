package SearchEngines

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"time"

	"github.com/goccy/go-json"
	"github.com/meilisearch/meilisearch-go"
)

func GetClient() *meilisearch.Client {
	meili := new(config.MeiliSearch).MeiliSearch()
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   meili.URL,
		APIKey: meili.ApiKey,
	})
	return client
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
		zlog.Error(err.Error())
	}
	if searchResponse.EstimatedTotalHits == 0 {
		return []interface{}{}
	}
	return searchResponse.Hits
}

func docSyncTask() {
	for {
		client := GetClient()
		var articleSearchDTOs []model.ArticleSearchDTO
		err := ormInit.GetEngine().SQL("select id, article_title, SUBSTR(article_content, 1, 500) AS " +
			"article_content, is_delete, status from t_article where is_delete = 0 and status = 1").Find(&articleSearchDTOs)
		if err != nil {
			zlog.Error(err.Error())
			continue
		}
		var docs []map[string]interface{}
		data, _ := json.Marshal(articleSearchDTOs)
		err = json.Unmarshal(data, &docs)
		if err != nil {
			zlog.Error(err.Error())
			continue
		}
		_, err = client.Index("articles").UpdateDocuments(docs)
		if err != nil {
			zlog.Error(err.Error())
			continue
		}
		zlog.Info("-----docs completed with synchronization-----")
		time.Sleep(time.Minute * 10)
	}
}

func init() {
	client := GetClient()
	index, err1 := client.GetIndex("articles")
	if index == nil && err1 != nil {
		var articleSearchDTOs []model.ArticleSearchDTO
		err := ormInit.GetEngine().SQL("select id, article_title, SUBSTR(article_content, 1, 500) AS " +
			"article_content, is_delete, status from t_article where is_delete = 0 and status = 1").Find(&articleSearchDTOs)
		if err != nil {
			zlog.Error(err.Error())
		}
		var docs []map[string]interface{}
		data, _ := json.Marshal(articleSearchDTOs)
		err = json.Unmarshal(data, &docs)
		if err != nil {
			zlog.Error(err.Error())
		}
		_, err = client.Index("articles").AddDocuments(docs)
		if err != nil {
			zlog.Error(err.Error())
		}
	}
	// 开启一个协程执行search同步工作
	go docSyncTask()
}
