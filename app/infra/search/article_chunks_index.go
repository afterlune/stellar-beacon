package search

import (
	"context"
	stderrors "errors"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/infra/config"

	"github.com/meilisearch/meilisearch-go"
)

const (
	ArticleChunksIndexPrefix     = "article_chunks"
	ArticleChunksIndexPrimaryKey = "id"
	maxArticleChunksIndexVersion = 31
	maxEmbeddingBatchSize        = 1024
)

var (
	articleChunksSearchableFields = []string{ArticleChunkTitleField, ArticleChunkTextField}
	articleChunksFilterableFields = []interface{}{
		ArticleChunkArticleIDField,
		ArticleChunkCategoryField,
		ArticleChunkTagsField,
		ArticleChunkPublishedAtField,
		ArticleChunkPublishedAtUnixField,
		ArticleChunkStatusField,
		ArticleChunkIsDeleteField,
	}
)

var articleChunksIndexVersionPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,30}$`)

// ArticleChunksIndexSpec is the immutable identity of one embedding index.
// Changing model version or dimension must produce a new spec and therefore a
// new index UID; vectors from different embedding contracts must never mix.
type ArticleChunksIndexSpec struct {
	UID                string
	PrimaryKey         string
	Provider           string
	Model              string
	ModelVersion       string
	IndexVersion       string
	Dimension          int
	EmbeddingBatchSize int
}

// ValidateArticleChunksIndexMigration checks the operator-supplied old and
// target contracts before a reindex starts. There is no reliable way to infer
// the old embedding contract from Meilisearch alone, so callers must provide
// it explicitly. A changed model, model version, provider, or dimension may
// never reuse the same index UID.
func ValidateArticleChunksIndexMigration(previous, target ArticleChunksIndexSpec) error {
	if err := previous.Validate(); err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "search.article_chunks.previous_spec", err)
	}
	if err := target.Validate(); err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "search.article_chunks.target_spec", err)
	}
	if previous.UID == target.UID && !sameEmbeddingContract(previous, target) {
		return apperrors.Conflict("search.article_chunks.migration", "embedding contract changes require a new index UID")
	}
	return nil
}

func sameEmbeddingContract(left, right ArticleChunksIndexSpec) bool {
	return left.Provider == right.Provider &&
		left.Model == right.Model &&
		left.ModelVersion == right.ModelVersion &&
		left.Dimension == right.Dimension
}

// EnsureArticleChunksIndex creates the requested versioned index only when it
// is explicitly called. It never changes an existing index's primary key and
// does not run from application startup. The returned task is asynchronous;
// callers that need an operational index must wait for that task before
// writing documents.
func EnsureArticleChunksIndex(ctx context.Context, client meilisearch.ServiceManager, spec ArticleChunksIndexSpec) (*meilisearch.TaskInfo, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, apperrors.Unavailable("search.article_chunks.ensure", fmt.Errorf("Meilisearch client is not configured"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	existing, err := client.GetIndexWithContext(ctx, spec.UID)
	if err == nil {
		if existing == nil {
			return nil, apperrors.Unavailable("search.article_chunks.inspect", fmt.Errorf("Meilisearch returned an empty index response"))
		}
		if existing.PrimaryKey != spec.PrimaryKey {
			return nil, apperrors.Conflict("search.article_chunks.ensure", fmt.Sprintf("index %q has primary key %q, want %q", spec.UID, existing.PrimaryKey, spec.PrimaryKey))
		}
		return nil, nil
	}
	if !isMeiliNotFound(err) {
		return nil, apperrors.Unavailable("search.article_chunks.inspect", err)
	}

	task, err := client.CreateIndexWithContext(ctx, &meilisearch.IndexConfig{
		Uid:        spec.UID,
		PrimaryKey: spec.PrimaryKey,
	})
	if err == nil {
		if task == nil {
			return nil, apperrors.Unavailable("search.article_chunks.create", fmt.Errorf("Meilisearch returned an empty create task"))
		}
		return task, nil
	}
	// Another explicit provisioning attempt may win the race between GET and
	// POST. Re-read only for that conflict; never hide unrelated Meili errors.
	if !isMeiliConflict(err) {
		return nil, apperrors.Unavailable("search.article_chunks.create", err)
	}
	existing, inspectErr := client.GetIndexWithContext(ctx, spec.UID)
	if inspectErr != nil {
		return nil, apperrors.Unavailable("search.article_chunks.inspect", inspectErr)
	}
	if existing == nil {
		return nil, apperrors.Unavailable("search.article_chunks.inspect", fmt.Errorf("Meilisearch returned an empty index response"))
	}
	if existing.PrimaryKey != spec.PrimaryKey {
		return nil, apperrors.Conflict("search.article_chunks.ensure", fmt.Sprintf("index %q has primary key %q, want %q", spec.UID, existing.PrimaryKey, spec.PrimaryKey))
	}
	return nil, nil
}

// ProvisionArticleChunksIndex performs the complete explicit Meilisearch
// provisioning step required by the worker. It is intentionally separate
// from application startup: callers must invoke it in a release window after
// reviewing the target versioned UID. The articleId filter is required for
// deleting all old chunks when an article becomes private or gets shorter.
func ProvisionArticleChunksIndex(ctx context.Context, client meilisearch.ServiceManager, spec ArticleChunksIndexSpec) ([]*meilisearch.TaskInfo, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, apperrors.Unavailable("search.article_chunks.provision", fmt.Errorf("Meilisearch client is not configured"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	createTask, err := EnsureArticleChunksIndex(ctx, client, spec)
	if err != nil {
		return nil, err
	}
	index := client.Index(spec.UID)
	searchable := append([]string(nil), articleChunksSearchableFields...)
	filterable := append([]interface{}(nil), articleChunksFilterableFields...)
	searchableTask, err := index.UpdateSearchableAttributesWithContext(ctx, &searchable)
	if err != nil {
		return nil, apperrors.Unavailable("search.article_chunks.searchable_attributes", err)
	}
	if searchableTask == nil {
		return nil, apperrors.Unavailable("search.article_chunks.searchable_attributes", fmt.Errorf("Meilisearch returned an empty settings task"))
	}
	filterableTask, err := index.UpdateFilterableAttributesWithContext(ctx, &filterable)
	if err != nil {
		return nil, apperrors.Unavailable("search.article_chunks.filterable_attributes", err)
	}
	if filterableTask == nil {
		return nil, apperrors.Unavailable("search.article_chunks.filterable_attributes", fmt.Errorf("Meilisearch returned an empty settings task"))
	}
	tasks := make([]*meilisearch.TaskInfo, 0, 3)
	if createTask != nil {
		tasks = append(tasks, createTask)
	}
	if searchableTask != nil {
		tasks = append(tasks, searchableTask)
	}
	if filterableTask != nil {
		tasks = append(tasks, filterableTask)
	}
	return tasks, nil
}

func isMeiliNotFound(err error) bool {
	var meiliErr *meilisearch.Error
	return stderrors.As(err, &meiliErr) && meiliErr.StatusCode == http.StatusNotFound
}

func isMeiliConflict(err error) bool {
	var meiliErr *meilisearch.Error
	return stderrors.As(err, &meiliErr) && meiliErr.StatusCode == http.StatusConflict
}

func validateArticleChunksIndexSettings(ctx context.Context, client meilisearch.ServiceManager, uid string) error {
	if client == nil {
		return apperrors.Unavailable("search.article_chunks.swap.settings", fmt.Errorf("Meilisearch client is not configured"))
	}
	index := client.Index(uid)
	if index == nil {
		return apperrors.Unavailable("search.article_chunks.swap.settings", fmt.Errorf("Meilisearch index manager is not configured"))
	}
	settings, err := index.GetSettingsWithContext(ctx)
	if err != nil {
		return apperrors.Unavailable("search.article_chunks.swap.settings", err)
	}
	if settings == nil {
		return apperrors.Unavailable("search.article_chunks.swap.settings", fmt.Errorf("Meilisearch returned an empty settings response"))
	}
	expectedSearchable := append([]string(nil), articleChunksSearchableFields...)
	if !reflect.DeepEqual(settings.SearchableAttributes, expectedSearchable) {
		return apperrors.Conflict("search.article_chunks.swap.settings", "searchable attributes do not match the article chunk contract")
	}
	expectedFilterable := make([]string, 0, len(articleChunksFilterableFields))
	for _, field := range articleChunksFilterableFields {
		name, ok := field.(string)
		if !ok {
			return apperrors.Unavailable("search.article_chunks.swap.settings", fmt.Errorf("article chunk filterable field is not a string"))
		}
		expectedFilterable = append(expectedFilterable, name)
	}
	if !sameStringSet(settings.FilterableAttributes, expectedFilterable) {
		return apperrors.Conflict("search.article_chunks.swap.settings", "filterable attributes do not match the article chunk contract")
	}
	return nil
}

// sameStringSet compares filterable attributes as a set. Meilisearch may
// normalize the order of filterable attributes when it returns settings, and
// that order has no effect on filtering semantics. Sorting copies preserves
// the duplicate check while avoiding mutation of SDK-owned slices.
func sameStringSet(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualCopy := append([]string(nil), actual...)
	expectedCopy := append([]string(nil), expected...)
	sort.Strings(actualCopy)
	sort.Strings(expectedCopy)
	return reflect.DeepEqual(actualCopy, expectedCopy)
}

// ArticleChunksIndexUID returns a safe, versioned UID without touching
// Meilisearch. Index creation and migration are deliberately left to later
// explicit jobs so service startup cannot mutate an existing container.
func ArticleChunksIndexUID(version string) (string, error) {
	version = strings.ToLower(strings.TrimSpace(version))
	if version == "" {
		return "", apperrors.Invalid("search.article_chunks.index_version", "index version is required")
	}
	if len(version) > maxArticleChunksIndexVersion || !articleChunksIndexVersionPattern.MatchString(version) {
		return "", apperrors.Invalid("search.article_chunks.index_version", "index version must match [a-z0-9][a-z0-9_-]{0,30}")
	}
	return ArticleChunksIndexPrefix + "_" + version, nil
}

func NewArticleChunksIndexSpec(route config.AIModelRoute, embedding config.AIEmbeddingConfig) (ArticleChunksIndexSpec, error) {
	provider := strings.ToLower(strings.TrimSpace(route.Provider))
	model := strings.TrimSpace(route.Model)
	modelVersion := strings.TrimSpace(embedding.ModelVersion)
	if provider == "" {
		return ArticleChunksIndexSpec{}, apperrors.Invalid("search.article_chunks.provider", "embedding provider is required")
	}
	if model == "" {
		return ArticleChunksIndexSpec{}, apperrors.Invalid("search.article_chunks.model", "embedding model is required")
	}
	if modelVersion == "" {
		return ArticleChunksIndexSpec{}, apperrors.Invalid("search.article_chunks.model_version", "embedding model version is required")
	}
	if embedding.Dimension <= 0 {
		return ArticleChunksIndexSpec{}, apperrors.Invalid("search.article_chunks.dimension", "embedding dimension must be positive")
	}
	if embedding.BatchSize <= 0 || embedding.BatchSize > maxEmbeddingBatchSize {
		return ArticleChunksIndexSpec{}, apperrors.Invalid("search.article_chunks.batch_size", fmt.Sprintf("embedding batch size must be between 1 and %d", maxEmbeddingBatchSize))
	}
	uid, err := ArticleChunksIndexUID(embedding.IndexVersion)
	if err != nil {
		return ArticleChunksIndexSpec{}, err
	}
	spec := ArticleChunksIndexSpec{
		UID:                uid,
		PrimaryKey:         ArticleChunksIndexPrimaryKey,
		Provider:           provider,
		Model:              model,
		ModelVersion:       modelVersion,
		IndexVersion:       strings.ToLower(strings.TrimSpace(embedding.IndexVersion)),
		Dimension:          embedding.Dimension,
		EmbeddingBatchSize: embedding.BatchSize,
	}
	return spec, spec.Validate()
}

// Validate confirms that a spec is self-consistent before it is used for
// provisioning or document writes. Keeping this check on the value itself
// prevents callers from bypassing NewArticleChunksIndexSpec with a partially
// populated struct.
func (s ArticleChunksIndexSpec) Validate() error {
	if s.PrimaryKey != ArticleChunksIndexPrimaryKey {
		return apperrors.Invalid("search.article_chunks.primary_key", "article_chunks primary key must be id")
	}
	if strings.TrimSpace(s.Provider) == "" {
		return apperrors.Invalid("search.article_chunks.provider", "embedding provider is required")
	}
	if strings.TrimSpace(s.Model) == "" {
		return apperrors.Invalid("search.article_chunks.model", "embedding model is required")
	}
	if strings.TrimSpace(s.ModelVersion) == "" {
		return apperrors.Invalid("search.article_chunks.model_version", "embedding model version is required")
	}
	if s.Dimension <= 0 {
		return apperrors.Invalid("search.article_chunks.dimension", "embedding dimension must be positive")
	}
	if s.EmbeddingBatchSize <= 0 || s.EmbeddingBatchSize > maxEmbeddingBatchSize {
		return apperrors.Invalid("search.article_chunks.batch_size", fmt.Sprintf("embedding batch size must be between 1 and %d", maxEmbeddingBatchSize))
	}
	expectedUID, err := ArticleChunksIndexUID(s.IndexVersion)
	if err != nil {
		return err
	}
	if s.UID != expectedUID {
		return apperrors.Invalid("search.article_chunks.uid", "index UID does not match index version")
	}
	return nil
}

// ValidateVector prevents a worker from writing malformed or incompatible
// vectors to an index. NaN and Inf are not valid JSON numbers and can poison a
// batch even when the vector length is correct.
func (s ArticleChunksIndexSpec) ValidateVector(vector []float32) error {
	if s.Dimension <= 0 || len(vector) != s.Dimension {
		return apperrors.Invalid("search.article_chunks.vector", fmt.Sprintf("vector dimension is %d, want %d", len(vector), s.Dimension))
	}
	for index, value := range vector {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return apperrors.Invalid("search.article_chunks.vector", fmt.Sprintf("vector contains a non-finite value at position %d", index))
		}
	}
	return nil
}
