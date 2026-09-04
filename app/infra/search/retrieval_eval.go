package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	searchevaldata "benetnasch/app/infra/search/evaldata"
)

// RetrievalEvalK is the fixed cutoff used by the offline retrieval benchmark.
const RetrievalEvalK = 8

type RetrievalEvalCaseResult struct {
	ID               string  `json:"id"`
	Returned         int     `json:"returned"`
	Relevant         int     `json:"relevant"`
	RecallAt8        float64 `json:"recallAt8"`
	ReciprocalRank   float64 `json:"reciprocalRank"`
	NoResultExpected bool    `json:"noResultExpected"`
	NoResultCorrect  bool    `json:"noResultCorrect"`
}

type RetrievalEvalReport struct {
	K                int                       `json:"k"`
	Queries          int                       `json:"queries"`
	PositiveQueries  int                       `json:"positiveQueries"`
	NoResultQueries  int                       `json:"noResultQueries"`
	RecallAt8        float64                   `json:"recallAt8"`
	MRR              float64                   `json:"mrr"`
	NoResultAccuracy float64                   `json:"noResultAccuracy"`
	Cases            []RetrievalEvalCaseResult `json:"cases"`
}

// RetrievalLatencyCase records one end-to-end search duration. The measured
// duration includes the adapter and its configured dependencies, but never
// includes model first-token latency; that is reported by ai.RunMetricsObserver.
type RetrievalLatencyCase struct {
	ID       string        `json:"id"`
	Duration time.Duration `json:"duration"`
}

// RetrievalLatencyReport is an offline performance gate. Percentiles are
// calculated over the supplied fixed cases, so a production release must run
// the same evaluator against a representative isolated deployment before
// treating Pass as a release decision.
type RetrievalLatencyReport struct {
	Samples int                    `json:"samples"`
	P50     time.Duration          `json:"p50"`
	P95     time.Duration          `json:"p95"`
	Max     time.Duration          `json:"max"`
	Target  time.Duration          `json:"target"`
	Pass    bool                   `json:"pass"`
	Cases   []RetrievalLatencyCase `json:"cases"`
}

// EvaluateKnowledgeSearch evaluates a search adapter against supplied cases.
// Recall@8 and MRR are macro-averaged over positive cases. No-result accuracy
// is calculated only over cases explicitly marked as no-result. Hit IDs are
// de-duplicated in rank order before the top-eight cutoff is applied.
//
// A search error aborts evaluation; it is never converted into an empty result
// because doing so would falsely improve no-result accuracy.
func EvaluateKnowledgeSearch(ctx context.Context, searcher port.KnowledgeSearcher, cases []searchevaldata.Case) (RetrievalEvalReport, error) {
	report := RetrievalEvalReport{K: RetrievalEvalK}
	if ctx == nil {
		ctx = context.Background()
	}
	if searcher == nil {
		return report, apperrors.Unavailable("search.evaluation", fmt.Errorf("knowledge searcher is not configured"))
	}
	normalized, err := searchevaldata.NormalizeCases(cases)
	if err != nil {
		return report, apperrors.Invalid("search.evaluation.dataset", err.Error())
	}
	report.Queries = len(normalized)
	report.Cases = make([]RetrievalEvalCaseResult, 0, len(normalized))

	var recallSum float64
	var reciprocalRankSum float64
	noResultCorrect := 0
	for _, testCase := range normalized {
		if testCase.ExpectNoResult {
			report.NoResultQueries++
		} else {
			report.PositiveQueries++
		}

		hits, err := searcher.Search(ctx, port.KnowledgeQuery{
			Query: testCase.Query,
			Mode:  testCase.Mode,
			Limit: RetrievalEvalK,
		})
		if err != nil {
			return report, fmt.Errorf("retrieval evaluation case %q: %w", testCase.ID, err)
		}

		returnedIDs := uniqueHitIDs(hits, RetrievalEvalK)
		result := RetrievalEvalCaseResult{
			ID:               testCase.ID,
			Returned:         len(returnedIDs),
			NoResultExpected: testCase.ExpectNoResult,
		}
		if testCase.ExpectNoResult {
			result.NoResultCorrect = len(returnedIDs) == 0
			if result.NoResultCorrect {
				noResultCorrect++
			}
		} else {
			relevantIDs := make(map[string]struct{}, len(testCase.Expected))
			for _, expectedID := range testCase.Expected {
				relevantIDs[expectedID] = struct{}{}
			}
			for rank, hitID := range returnedIDs {
				if _, relevant := relevantIDs[hitID]; !relevant {
					continue
				}
				result.Relevant++
				if result.ReciprocalRank == 0 {
					result.ReciprocalRank = 1 / float64(rank+1)
				}
			}
			result.RecallAt8 = float64(result.Relevant) / float64(len(relevantIDs))
			recallSum += result.RecallAt8
			reciprocalRankSum += result.ReciprocalRank
		}
		report.Cases = append(report.Cases, result)
	}
	if report.PositiveQueries > 0 {
		report.RecallAt8 = recallSum / float64(report.PositiveQueries)
		report.MRR = reciprocalRankSum / float64(report.PositiveQueries)
	}
	if report.NoResultQueries > 0 {
		report.NoResultAccuracy = float64(noResultCorrect) / float64(report.NoResultQueries)
	}
	return report, nil
}

// EvaluateKnowledgeSearchDataset evaluates an adapter against the embedded
// fixed dataset and is the default offline evaluation entry point.
func EvaluateKnowledgeSearchDataset(ctx context.Context, searcher port.KnowledgeSearcher) (RetrievalEvalReport, error) {
	cases, err := searchevaldata.Load()
	if err != nil {
		return RetrievalEvalReport{K: RetrievalEvalK}, err
	}
	return EvaluateKnowledgeSearch(ctx, searcher, cases)
}

// EvaluateKnowledgeSearchLatency measures all cases in a fixed retrieval
// dataset and applies the supplied P95 target. Search errors abort the gate;
// an unavailable backend must never look like a fast empty response.
func EvaluateKnowledgeSearchLatency(ctx context.Context, searcher port.KnowledgeSearcher, cases []searchevaldata.Case, target time.Duration) (RetrievalLatencyReport, error) {
	report := RetrievalLatencyReport{Target: target}
	if target <= 0 {
		return report, apperrors.Invalid("search.evaluation.latency", "latency target must be positive")
	}
	if searcher == nil {
		return report, apperrors.Unavailable("search.evaluation.latency", fmt.Errorf("knowledge searcher is not configured"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	normalized, err := searchevaldata.NormalizeCases(cases)
	if err != nil {
		return report, apperrors.Invalid("search.evaluation.latency.dataset", err.Error())
	}
	report.Cases = make([]RetrievalLatencyCase, 0, len(normalized))
	durations := make([]time.Duration, 0, len(normalized))
	for _, testCase := range normalized {
		started := time.Now()
		if _, err := searcher.Search(ctx, port.KnowledgeQuery{Query: testCase.Query, Mode: testCase.Mode, Limit: RetrievalEvalK}); err != nil {
			return report, fmt.Errorf("retrieval latency evaluation case %q: %w", testCase.ID, err)
		}
		duration := time.Since(started)
		durations = append(durations, duration)
		report.Cases = append(report.Cases, RetrievalLatencyCase{ID: testCase.ID, Duration: duration})
	}
	report.Samples = len(durations)
	report.P50 = durationPercentile(durations, 50)
	report.P95 = durationPercentile(durations, 95)
	for _, duration := range durations {
		if duration > report.Max {
			report.Max = duration
		}
	}
	report.Pass = report.P95 <= target
	return report, nil
}

// EvaluateKnowledgeSearchDatasetLatency applies the default 300ms internal
// search target to the embedded fixed dataset. Model first-token latency is
// intentionally not included in this target.
func EvaluateKnowledgeSearchDatasetLatency(ctx context.Context, searcher port.KnowledgeSearcher) (RetrievalLatencyReport, error) {
	cases, err := searchevaldata.Load()
	if err != nil {
		return RetrievalLatencyReport{Target: 300 * time.Millisecond}, err
	}
	return EvaluateKnowledgeSearchLatency(ctx, searcher, cases, 300*time.Millisecond)
}

func uniqueHitIDs(hits []port.KnowledgeHit, limit int) []string {
	if limit <= 0 {
		return nil
	}
	ids := make([]string, 0, minInt(len(hits), limit))
	seen := make(map[string]struct{}, len(hits))
	for _, hit := range hits {
		id := strings.TrimSpace(hit.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if len(ids) == limit {
			break
		}
	}
	return ids
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func durationPercentile(values []time.Duration, percentile int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if percentile < 1 {
		percentile = 1
	}
	if percentile > 100 {
		percentile = 100
	}
	index := (len(sorted)*percentile+99)/100 - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
