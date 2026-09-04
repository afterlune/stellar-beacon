package evaldata

import (
	"testing"

	"benetnasch/app/domain/port"
)

func TestAggregateReviewOutcomeMetricsCalculatesApprovalAndDuplicateRates(t *testing.T) {
	metrics, err := AggregateReviewOutcomeMetrics([]port.AIReview{
		{Status: port.ReviewPending, Content: "候选一"},
		{Status: port.ReviewApproved, Content: "候选一"},
		{Status: port.ReviewPartiallyApproved, Content: "候选二"},
		{Status: port.ReviewRejected, Content: "候选三"},
		{Status: port.ReviewExpired, Content: "候选四"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Total != 5 || metrics.Pending != 1 || metrics.Decided != 4 || metrics.Accepted != 2 || metrics.DuplicateCandidates != 1 || metrics.DigestCandidates != 5 {
		t.Fatalf("review metrics = %#v", metrics)
	}
	if metrics.ApprovalRate != 0.5 || metrics.RejectionRate != 0.25 || metrics.ExpiryRate != 0.25 || metrics.DuplicateRate != 0.2 {
		t.Fatalf("review rates = %#v", metrics)
	}
}

func TestAggregateReviewOutcomeMetricsRejectsUnknownStatus(t *testing.T) {
	if _, err := AggregateReviewOutcomeMetrics([]port.AIReview{{Status: port.ReviewStatus("model_approved")}}); err == nil {
		t.Fatal("unknown review status was accepted")
	}
}

func TestAggregateWritingQualityAllowsUnannotatedCollectionBatch(t *testing.T) {
	cases, err := LoadWritingDataset()
	if err != nil {
		t.Fatal(err)
	}
	report, err := AggregateWritingQuality(cases, nil, []port.AIReview{{Status: port.ReviewPending, Content: "待评审候选"}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Human.TotalCases != len(cases) || report.Human.Complete || report.Human.Pass || report.Review.Pending != 1 {
		t.Fatalf("quality report = %#v", report)
	}
}
