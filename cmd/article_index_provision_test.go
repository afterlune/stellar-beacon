package cmd

import (
	"bytes"
	"strings"
	"testing"

	"benetnasch/app/infra/search"

	"github.com/meilisearch/meilisearch-go"
)

func TestRequireArticleIndexWriteAuthorization(t *testing.T) {
	if err := requireArticleIndexWriteAuthorization(false); err == nil {
		t.Fatal("provision should reject missing explicit write authorization")
	}
	if err := requireArticleIndexWriteAuthorization(true); err != nil {
		t.Fatalf("authorized provision rejected: %v", err)
	}
}

func TestWriteArticleIndexProvisionOutputContainsContractWithoutSecrets(t *testing.T) {
	spec := search.ArticleChunksIndexSpec{
		UID:                "article_chunks_v2",
		IndexVersion:       "v2",
		Provider:           "alibailian",
		Model:              "qwen3.7-text-embedding",
		ModelVersion:       "qwen3.7-text-embedding",
		Dimension:          1024,
		EmbeddingBatchSize: 32,
	}
	tasks := []*meilisearch.TaskInfo{{TaskUID: 12, IndexUID: spec.UID}}
	completed := []*meilisearch.Task{{TaskUID: 12, IndexUID: spec.UID}}
	var output bytes.Buffer
	if err := writeArticleIndexProvisionOutput(&output, spec, tasks, completed); err != nil {
		t.Fatalf("writeArticleIndexProvisionOutput() error = %v", err)
	}
	for _, expected := range []string{"article_chunks_v2", "alibailian", "qwen3.7-text-embedding", "\"taskUid\": 12"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output = %s, missing %q", output.String(), expected)
		}
	}
	for _, forbidden := range []string{"apiKey", "MEILI_MASTER_KEY", "secret", "password"} {
		if strings.Contains(strings.ToLower(output.String()), strings.ToLower(forbidden)) {
			t.Fatalf("output = %s, contains secret field %q", output.String(), forbidden)
		}
	}
}

func TestWriteArticleIndexProvisionOutputRejectsNilWriter(t *testing.T) {
	if err := writeArticleIndexProvisionOutput(nil, search.ArticleChunksIndexSpec{}, nil, nil); err == nil {
		t.Fatal("expected nil output error")
	}
}

func TestNewArticleIndexSwapCurrentSpecRequiresMatchingUID(t *testing.T) {
	spec, err := newArticleIndexSwapCurrentSpec("article_chunks_v1", "v1", "legacy", "legacy-embedding", "legacy-v1", 1024, 32)
	if err != nil {
		t.Fatalf("newArticleIndexSwapCurrentSpec() error = %v", err)
	}
	if spec.UID != "article_chunks_v1" || spec.PrimaryKey != "id" {
		t.Fatalf("current spec = %+v", spec)
	}
	if _, err := newArticleIndexSwapCurrentSpec("article_chunks_v2", "v1", "legacy", "legacy-embedding", "legacy-v1", 1024, 32); err == nil {
		t.Fatal("expected current UID/version mismatch")
	}
}

func TestWriteArticleIndexSwapOutputContainsOnlyCutoverEvidence(t *testing.T) {
	plan, err := search.NewArticleChunksIndexSwapPlan(
		search.ArticleChunksIndexSpec{UID: "article_chunks_v1", IndexVersion: "v1", PrimaryKey: "id", Provider: "legacy", Model: "legacy-embedding", ModelVersion: "legacy-v1", Dimension: 1024, EmbeddingBatchSize: 32},
		search.ArticleChunksIndexSpec{UID: "article_chunks_v2", IndexVersion: "v2", PrimaryKey: "id", Provider: "alibailian", Model: "qwen3.7-text-embedding", ModelVersion: "qwen3.7-text-embedding", Dimension: 1024, EmbeddingBatchSize: 32},
	)
	if err != nil {
		t.Fatalf("NewArticleChunksIndexSwapPlan() error = %v", err)
	}
	var output bytes.Buffer
	if err := writeArticleIndexSwapOutput(&output, plan, &meilisearch.TaskInfo{TaskUID: 42}, []*meilisearch.Task{{UID: 42, IndexUID: "article_chunks_v2"}}); err != nil {
		t.Fatalf("writeArticleIndexSwapOutput() error = %v", err)
	}
	for _, expected := range []string{"article_chunks_v1", "article_chunks_v2", "\"swapTaskUid\": 42", "\"completedTaskUid\": 42"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("output = %s, missing %q", output.String(), expected)
		}
	}
	for _, forbidden := range []string{"apiKey", "MEILI_MASTER_KEY", "password", "secret"} {
		if strings.Contains(strings.ToLower(output.String()), strings.ToLower(forbidden)) {
			t.Fatalf("output = %s, contains forbidden field %q", output.String(), forbidden)
		}
	}
}

func TestWriteArticleIndexSwapOutputRejectsUnfinishedTask(t *testing.T) {
	plan := search.ArticleChunksIndexSwapPlan{
		Current:   search.ArticleChunksIndexSpec{UID: "article_chunks_v1", IndexVersion: "v1", PrimaryKey: "id", Provider: "legacy", Model: "legacy-embedding", ModelVersion: "legacy-v1", Dimension: 1024, EmbeddingBatchSize: 32},
		Candidate: search.ArticleChunksIndexSpec{UID: "article_chunks_v2", IndexVersion: "v2", PrimaryKey: "id", Provider: "alibailian", Model: "qwen3.7-text-embedding", ModelVersion: "qwen3.7-text-embedding", Dimension: 1024, EmbeddingBatchSize: 32},
	}
	if err := writeArticleIndexSwapOutput(&bytes.Buffer{}, plan, &meilisearch.TaskInfo{TaskUID: 42}, nil); err == nil {
		t.Fatal("expected unfinished swap task to be rejected")
	}
}
