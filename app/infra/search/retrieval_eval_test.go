package search

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	searchevaldata "benetnasch/app/infra/search/evaldata"
)

type retrievalEvalSearcher struct {
	hitsByQuery map[string][]port.KnowledgeHit
	err         error
	queries     []port.KnowledgeQuery
}

func (f *retrievalEvalSearcher) Search(_ context.Context, query port.KnowledgeQuery) ([]port.KnowledgeHit, error) {
	f.queries = append(f.queries, query)
	if f.err != nil {
		return nil, f.err
	}
	return f.hitsByQuery[query.Query], nil
}

func TestEvaluateKnowledgeSearchCalculatesMetrics(t *testing.T) {
	cases := []searchevaldata.Case{
		{ID: "first", Query: "first", Expected: []string{"doc-a"}},
		{ID: "second", Query: "second", Expected: []string{"doc-b", "doc-c"}},
		{ID: "empty", Query: "empty", ExpectNoResult: true},
	}
	searcher := &retrievalEvalSearcher{hitsByQuery: map[string][]port.KnowledgeHit{
		"first":  {{ID: "doc-a"}},
		"second": {{ID: "doc-x"}, {ID: "doc-b"}, {ID: "doc-c"}},
		"empty":  nil,
	}}

	report, err := EvaluateKnowledgeSearch(context.Background(), searcher, cases)
	if err != nil {
		t.Fatal(err)
	}
	if report.K != RetrievalEvalK || report.Queries != 3 || report.PositiveQueries != 2 || report.NoResultQueries != 1 {
		t.Fatalf("report counts = %+v", report)
	}
	assertFloat(t, report.RecallAt8, 1)
	assertFloat(t, report.MRR, 0.75)
	assertFloat(t, report.NoResultAccuracy, 1)
	if len(report.Cases) != 3 || report.Cases[1].Relevant != 2 || report.Cases[1].Returned != 3 {
		t.Fatalf("case results = %+v", report.Cases)
	}
	for index, query := range searcher.queries {
		if query.Limit != RetrievalEvalK {
			t.Fatalf("query %d limit = %d, want %d", index, query.Limit, RetrievalEvalK)
		}
	}
}

func TestEvaluateKnowledgeSearchUsesUniqueTopEightIDs(t *testing.T) {
	hits := make([]port.KnowledgeHit, 0, 10)
	hits = append(hits, port.KnowledgeHit{ID: "irrelevant"})
	hits = append(hits, port.KnowledgeHit{ID: "doc-a"})
	hits = append(hits, port.KnowledgeHit{ID: "doc-a"})
	for index := 0; index < 7; index++ {
		hits = append(hits, port.KnowledgeHit{ID: "other-" + string(rune('a'+index))})
	}
	hits = append(hits, port.KnowledgeHit{ID: "doc-b"})
	searcher := &retrievalEvalSearcher{hitsByQuery: map[string][]port.KnowledgeHit{"ranked": hits}}
	cases := []searchevaldata.Case{
		{ID: "ranked", Query: "ranked", Expected: []string{"doc-a", "doc-b"}},
		{ID: "empty", Query: "empty", ExpectNoResult: true},
	}

	report, err := EvaluateKnowledgeSearch(context.Background(), searcher, cases)
	if err != nil {
		t.Fatal(err)
	}
	if report.Cases[0].Returned != RetrievalEvalK || report.Cases[0].Relevant != 1 {
		t.Fatalf("top-eight case = %+v", report.Cases[0])
	}
	assertFloat(t, report.Cases[0].RecallAt8, 0.5)
	assertFloat(t, report.Cases[0].ReciprocalRank, 0.5)
}

func TestEvaluateKnowledgeSearchNoResultIsFalseForUnexpectedHit(t *testing.T) {
	searcher := &retrievalEvalSearcher{hitsByQuery: map[string][]port.KnowledgeHit{
		"no-result": {{ID: "unexpected"}},
		"empty":     nil,
	}}
	cases := []searchevaldata.Case{
		{ID: "no-result", Query: "no-result", ExpectNoResult: true},
		{ID: "empty", Query: "empty", ExpectNoResult: true},
		{ID: "positive", Query: "positive", Expected: []string{"doc"}},
	}

	report, err := EvaluateKnowledgeSearch(context.Background(), searcher, cases)
	if err != nil {
		t.Fatal(err)
	}
	assertFloat(t, report.NoResultAccuracy, 0.5)
	if report.Cases[0].NoResultCorrect || !report.Cases[1].NoResultCorrect {
		t.Fatalf("no-result cases = %+v", report.Cases)
	}
}

func TestEvaluateKnowledgeSearchAbortsOnSearchError(t *testing.T) {
	searchErr := errors.New("search backend unavailable")
	searcher := &retrievalEvalSearcher{err: searchErr}
	report, err := EvaluateKnowledgeSearch(context.Background(), searcher, []searchevaldata.Case{
		{ID: "positive", Query: "positive", Expected: []string{"doc"}},
		{ID: "empty", Query: "empty", ExpectNoResult: true},
	})
	if err == nil || !errors.Is(err, searchErr) {
		t.Fatalf("error = %v, want wrapped search error", err)
	}
	if len(report.Cases) != 0 {
		t.Fatalf("failed evaluation returned case results: %+v", report.Cases)
	}
}

func TestEvaluateKnowledgeSearchRejectsInvalidInputs(t *testing.T) {
	if _, err := EvaluateKnowledgeSearch(context.Background(), nil, nil); !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("nil searcher error kind = %v, want unavailable", apperrors.KindOf(err))
	}
	searcher := &retrievalEvalSearcher{}
	if _, err := EvaluateKnowledgeSearch(context.Background(), searcher, []searchevaldata.Case{
		{ID: "positive", Query: "positive", Expected: []string{"doc"}},
	}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("invalid dataset error kind = %v, want validation", apperrors.KindOf(err))
	}
}

func TestEvaluateKnowledgeSearchDatasetUsesEmbeddedCases(t *testing.T) {
	cases, err := searchevaldata.Load()
	if err != nil {
		t.Fatal(err)
	}
	hitsByQuery := make(map[string][]port.KnowledgeHit, len(cases))
	for _, testCase := range cases {
		if testCase.ExpectNoResult {
			hitsByQuery[testCase.Query] = nil
			continue
		}
		hitsByQuery[testCase.Query] = []port.KnowledgeHit{{ID: testCase.Expected[0]}}
	}

	report, err := EvaluateKnowledgeSearchDataset(context.Background(), &retrievalEvalSearcher{hitsByQuery: hitsByQuery})
	if err != nil {
		t.Fatal(err)
	}
	if report.Queries != len(cases) || len(report.Cases) != len(cases) {
		t.Fatalf("dataset report = %+v", report)
	}
	assertFloat(t, report.RecallAt8, 1)
	assertFloat(t, report.MRR, 1)
	assertFloat(t, report.NoResultAccuracy, 1)
}

type retrievalLatencySearcher struct {
	delay time.Duration
}

func (s retrievalLatencySearcher) Search(ctx context.Context, _ port.KnowledgeQuery) ([]port.KnowledgeHit, error) {
	timer := time.NewTimer(s.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestEvaluateKnowledgeSearchLatencySeparatesSearchP95FromModelTiming(t *testing.T) {
	cases := []searchevaldata.Case{
		{ID: "positive", Query: "positive", Expected: []string{"doc"}},
		{ID: "empty", Query: "empty", ExpectNoResult: true},
	}
	report, err := EvaluateKnowledgeSearchLatency(context.Background(), retrievalLatencySearcher{delay: time.Millisecond}, cases, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if report.Samples != 2 || report.P95 <= 0 || report.P95 > report.Target || !report.Pass || len(report.Cases) != 2 {
		t.Fatalf("latency report = %+v", report)
	}

	tooSmall, err := EvaluateKnowledgeSearchLatency(context.Background(), retrievalLatencySearcher{delay: time.Millisecond}, cases, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	if tooSmall.Pass {
		t.Fatalf("latency gate unexpectedly passed: %+v", tooSmall)
	}
}

func TestEvaluateKnowledgeSearchLatencyRejectsInvalidTarget(t *testing.T) {
	if _, err := EvaluateKnowledgeSearchLatency(context.Background(), retrievalLatencySearcher{}, nil, 0); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("invalid latency target error = %v, want validation", err)
	}
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("value = %v, want %v", got, want)
	}
}
