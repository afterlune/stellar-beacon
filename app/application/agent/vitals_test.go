package agent

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/port"
	"context"
	stderrors "errors"
	"testing"
	"time"
)

type vitalsSiteRepository struct{}

func (vitalsSiteRepository) CountArticles(context.Context) (int64, error)   { return 12, nil }
func (vitalsSiteRepository) CountCategories(context.Context) (int64, error) { return 3, nil }
func (vitalsSiteRepository) CountTags(context.Context) (int64, error)       { return 8, nil }
func (vitalsSiteRepository) CountTalks(context.Context) (int64, error)      { return 5, nil }
func (vitalsSiteRepository) CountRecentContent(context.Context, time.Time) (int64, error) {
	return 4, nil
}

func (vitalsSiteRepository) CountComments(context.Context, int) (int64, error) { return 0, nil }
func (vitalsSiteRepository) CountUsers(context.Context) (int64, error)         { return 0, nil }
func (vitalsSiteRepository) ListUniqueViews(context.Context, string, string) ([]port.UniqueView, error) {
	return nil, nil
}
func (vitalsSiteRepository) ListArticleRank(context.Context, []int) ([]port.ArticleRank, error) {
	return nil, nil
}
func (vitalsSiteRepository) GetWebsiteConfig(context.Context) (string, error)  { return "{}", nil }
func (vitalsSiteRepository) UpdateWebsiteConfig(context.Context, string) error { return nil }
func (vitalsSiteRepository) GetAbout(context.Context, int) (string, error)     { return "", nil }
func (vitalsSiteRepository) UpdateAbout(context.Context, int, string) error    { return nil }

type vitalsCache struct {
	port.Cache
	views    map[string]string
	areas    map[string]string
	getErr   error
	areasErr error
}

func (c vitalsCache) Get(context.Context, string) (string, error) {
	if c.getErr != nil {
		return "", c.getErr
	}
	if value, ok := c.views[support.BlogViewsCount]; ok {
		return value, nil
	}
	return "", port.ErrCacheMiss
}

func (c vitalsCache) HGetAll(context.Context, string) (map[string]string, error) {
	if c.areasErr != nil {
		return nil, c.areasErr
	}
	return c.areas, nil
}

func TestBasicVitalsAggregatesOnlyPublicCounters(t *testing.T) {
	policy, err := NewRhythmPolicy(DefaultRhythmSettings())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 29, 19, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	provider, err := NewBasicVitalsProvider(VitalsDeps{
		Site:   vitalsSiteRepository{},
		Cache:  vitalsCache{views: map[string]string{support.BlogViewsCount: "42"}, areas: map[string]string{"内网IP": "7", "上海": "5"}},
		Rhythm: policy,
		Now:    func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != port.AgentRhythmDusk || got.Status != "dusk" || got.Emotion != "reflective" {
		t.Fatalf("rhythm vitals = %+v", got)
	}
	if got.ContentCount != 17 || got.RecentContentCount != 4 || got.UniqueVisitorCount != 12 || got.ViewCount != 42 || got.ArticleCount != 12 {
		t.Fatalf("aggregate vitals = %+v", got)
	}
}

func TestBasicVitalsPreservesContextErrorsFromCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	provider, err := NewBasicVitalsProvider(VitalsDeps{
		Site:  vitalsSiteRepository{},
		Cache: vitalsCache{getErr: context.Canceled},
		Now:   time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Snapshot(ctx); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("Snapshot() view error = %v, want context canceled", err)
	}

	provider, err = NewBasicVitalsProvider(VitalsDeps{
		Site:  vitalsSiteRepository{},
		Cache: vitalsCache{areasErr: context.DeadlineExceeded},
		Now:   time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Snapshot(context.Background()); !stderrors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Snapshot() visitor error = %v, want deadline exceeded", err)
	}
}
