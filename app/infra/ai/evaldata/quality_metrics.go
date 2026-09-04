package evaldata

import (
	"fmt"
	"strings"

	"benetnasch/app/domain/port"
)

// ReviewOutcomeMetrics is the operational part of the writing-quality
// report. ApprovalRate and RejectionRate use decided reviews as their
// denominator; pending reviews are deliberately not treated as a rejection.
// DuplicateRate is the number of repeated candidates after the first one for
// each content digest divided by candidates that have a usable digest.
type ReviewOutcomeMetrics struct {
	Total               int     `json:"total"`
	Pending             int     `json:"pending"`
	Approved            int     `json:"approved"`
	PartiallyApproved   int     `json:"partiallyApproved"`
	Rejected            int     `json:"rejected"`
	Expired             int     `json:"expired"`
	Decided             int     `json:"decided"`
	Accepted            int     `json:"accepted"`
	DigestCandidates    int     `json:"digestCandidates"`
	DuplicateCandidates int     `json:"duplicateCandidates"`
	ApprovalRate        float64 `json:"approvalRate"`
	RejectionRate       float64 `json:"rejectionRate"`
	ExpiryRate          float64 `json:"expiryRate"`
	DuplicateRate       float64 `json:"duplicateRate"`
}

// WritingQualityReport combines the human writing rubric with review
// outcomes. Human scoring remains optional while a batch is being collected;
// when scores are present they are validated by AggregateWritingScores and
// retain its two-rater completeness/pass gate.
type WritingQualityReport struct {
	Human  WritingEvaluationReport `json:"human"`
	Review ReviewOutcomeMetrics    `json:"review"`
}

// AggregateWritingQuality calculates both the human quality report and the
// review approval/duplicate metrics without reading production state. The
// caller may pass no scores during annotation collection, but the writing
// cases are still validated and counted.
func AggregateWritingQuality(cases []WritingCase, scores []HumanWritingScore, reviews []port.AIReview) (WritingQualityReport, error) {
	normalizedCases, err := NormalizeWritingCases(cases)
	if err != nil {
		return WritingQualityReport{}, err
	}
	review, err := AggregateReviewOutcomeMetrics(reviews)
	if err != nil {
		return WritingQualityReport{}, err
	}
	report := WritingQualityReport{
		Review: review,
		Human:  WritingEvaluationReport{TotalCases: len(normalizedCases)},
	}
	if len(scores) > 0 {
		human, err := AggregateWritingScores(normalizedCases, scores)
		if err != nil {
			return WritingQualityReport{}, err
		}
		report.Human = human
	}
	return report, nil
}

// AggregateReviewOutcomeMetrics calculates bounded, deterministic quality
// signals from review records. It does not infer publication success from a
// model response; publication is an independent review state-machine result.
func AggregateReviewOutcomeMetrics(reviews []port.AIReview) (ReviewOutcomeMetrics, error) {
	metrics := ReviewOutcomeMetrics{Total: len(reviews)}
	digests := make(map[string]int, len(reviews))
	for index, review := range reviews {
		switch review.Status {
		case port.ReviewPending:
			metrics.Pending++
		case port.ReviewApproved:
			metrics.Approved++
		case port.ReviewPartiallyApproved:
			metrics.PartiallyApproved++
		case port.ReviewRejected:
			metrics.Rejected++
		case port.ReviewExpired:
			metrics.Expired++
		default:
			return ReviewOutcomeMetrics{}, fmt.Errorf("review %d has invalid status %q", index, review.Status)
		}
		digest := strings.ToLower(strings.TrimSpace(review.ContentDigest))
		if digest == "" && strings.TrimSpace(review.Content) != "" {
			digest = port.ReviewContentDigest(review.Content)
		}
		if digest != "" {
			metrics.DigestCandidates++
			digests[digest]++
		}
	}
	metrics.Decided = metrics.Approved + metrics.PartiallyApproved + metrics.Rejected + metrics.Expired
	metrics.Accepted = metrics.Approved + metrics.PartiallyApproved
	metrics.DuplicateCandidates = duplicateCount(digests)
	metrics.ApprovalRate = fraction(metrics.Accepted, metrics.Decided)
	metrics.RejectionRate = fraction(metrics.Rejected, metrics.Decided)
	metrics.ExpiryRate = fraction(metrics.Expired, metrics.Decided)
	metrics.DuplicateRate = fraction(metrics.DuplicateCandidates, metrics.DigestCandidates)
	return metrics, nil
}

func duplicateCount(digests map[string]int) int {
	duplicates := 0
	for _, count := range digests {
		if count > 1 {
			duplicates += count - 1
		}
	}
	return duplicates
}

func fraction(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
