package port

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewArticleIndexJobPayloadIsMetadataOnlyAndStable(t *testing.T) {
	when := time.Date(2026, 8, 28, 12, 34, 56, 0, time.FixedZone("CST", 8*60*60))
	payload, err := NewArticleIndexJobPayload(42, ArticleIndexUpsert, ArticleUpdated, PublicArticleStatus, 0, when)
	if err != nil {
		t.Fatalf("NewArticleIndexJobPayload() error = %v", err)
	}
	var got ArticleIndexJobPayload
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got.SchemaVersion != 1 || got.ArticleID != 42 || got.Action != ArticleIndexUpsert || got.Event != ArticleUpdated {
		t.Fatalf("unexpected payload: %+v", got)
	}
	if got.Status != PublicArticleStatus || got.IsDelete != 0 || got.OccurredAt.Location() != time.UTC {
		t.Fatalf("unexpected visibility or time: %+v", got)
	}
	if string(payload) == "" || containsJSONKey(payload, "articleContent") {
		t.Fatalf("payload unexpectedly contains article content: %s", payload)
	}
}

func TestNewArticleIndexJobPayloadRejectsInvalidValues(t *testing.T) {
	when := time.Now()
	tests := []struct {
		name      string
		articleID int
		action    ArticleIndexAction
		event     ArticleLifecycleEvent
		isDelete  int
	}{
		{name: "article id", articleID: 0, action: ArticleIndexDelete, event: ArticleDeleted},
		{name: "action", articleID: 1, action: "unknown", event: ArticleUpdated},
		{name: "event", articleID: 1, action: ArticleIndexDelete, event: "unknown"},
		{name: "delete flag", articleID: 1, action: ArticleIndexDelete, event: ArticleDeleted, isDelete: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewArticleIndexJobPayload(test.articleID, test.action, test.event, PublicArticleStatus, test.isDelete, when); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func containsJSONKey(payload []byte, key string) bool {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil {
		return false
	}
	_, ok := object[key]
	return ok
}
