package rag

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/ai/evaldata"
)

// RAGEvalCaseResult reports deterministic safety/provenance checks for one
// RAG response. Groundedness is an automated proxy: it requires a non-empty
// answer and correct citations, but it is not a replacement for human review.
type RAGEvalCaseResult struct {
	ID                string  `json:"id"`
	Grounded          bool    `json:"grounded"`
	CitationCorrect   bool    `json:"citationCorrect"`
	RefusalCorrect    bool    `json:"refusalCorrect"`
	ExpectedRefusal   bool    `json:"expectedRefusal"`
	CitationPrecision float64 `json:"citationPrecision"`
	CorrectCitations  int     `json:"correctCitations"`
	ReturnedCitations int     `json:"returnedCitations"`
}

// RAGEvaluationReport is intentionally model-agnostic so it can be used with
// a fake gateway in CI and with a configured provider in an explicit eval run.
type RAGEvaluationReport struct {
	TotalCases          int                 `json:"totalCases"`
	PositiveCases       int                 `json:"positiveCases"`
	RefusalCases        int                 `json:"refusalCases"`
	Groundedness        float64             `json:"groundedness"`
	CitationCorrectness float64             `json:"citationCorrectness"`
	RefusalAccuracy     float64             `json:"refusalAccuracy"`
	Cases               []RAGEvalCaseResult `json:"cases"`
}

// EvaluateRAG runs the fixed, explicit evaluation cases against a RAG port.
// A gateway error aborts the report instead of being counted as a refusal.
func EvaluateRAG(ctx context.Context, gateway port.RAGGateway, cases []evaldata.RAGCase) (RAGEvaluationReport, error) {
	report := RAGEvaluationReport{}
	if gateway == nil {
		return report, errors.Unavailable("agent.rag.evaluation", fmt.Errorf("RAG gateway is not configured"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	normalized, err := evaldata.NormalizeRAGCases(cases)
	if err != nil {
		return report, errors.Invalid("agent.rag.evaluation.dataset", err.Error())
	}
	report.TotalCases = len(normalized)
	report.Cases = make([]RAGEvalCaseResult, 0, len(normalized))
	grounded, citationCorrect, refusalCorrect := 0, 0, 0
	for _, testCase := range normalized {
		response, err := gateway.Generate(ctx, port.RAGRequest{Query: testCase.Query, Mode: testCase.Mode})
		if err != nil {
			return report, fmt.Errorf("RAG evaluation case %q: %w", testCase.ID, err)
		}
		if testCase.ExpectRefusal {
			report.RefusalCases++
		} else {
			report.PositiveCases++
		}
		result := evaluateRAGCase(testCase, response)
		if result.Grounded {
			grounded++
		}
		if result.CitationCorrect {
			citationCorrect++
		}
		if testCase.ExpectRefusal && result.RefusalCorrect {
			refusalCorrect++
		}
		report.Cases = append(report.Cases, result)
	}
	if report.TotalCases > 0 {
		report.Groundedness = float64(grounded) / float64(report.TotalCases)
		report.CitationCorrectness = float64(citationCorrect) / float64(report.TotalCases)
	}
	if report.RefusalCases > 0 {
		report.RefusalAccuracy = float64(refusalCorrect) / float64(report.RefusalCases)
	}
	return report, nil
}

// EvaluateEmbeddedRAG uses the repository's fixed offline dataset.
func EvaluateEmbeddedRAG(ctx context.Context, gateway port.RAGGateway) (RAGEvaluationReport, error) {
	cases, err := evaldata.LoadRAGCases()
	if err != nil {
		return RAGEvaluationReport{}, errors.Invalid("agent.rag.evaluation.dataset", err.Error())
	}
	return EvaluateRAG(ctx, gateway, cases)
}

func evaluateRAGCase(testCase evaldata.RAGCase, response port.RAGResponse) RAGEvalCaseResult {
	result := RAGEvalCaseResult{
		ID:                testCase.ID,
		ExpectedRefusal:   testCase.ExpectRefusal,
		ReturnedCitations: len(response.Citations),
	}
	expected := make(map[int]struct{}, len(testCase.ExpectedArticleIDs))
	for _, articleID := range testCase.ExpectedArticleIDs {
		expected[articleID] = struct{}{}
	}
	for _, citation := range response.Citations {
		if _, ok := expected[citation.ArticleID]; !ok || !validCitation(citation) {
			continue
		}
		result.CorrectCitations++
	}
	if result.ReturnedCitations > 0 {
		result.CitationPrecision = float64(result.CorrectCitations) / float64(result.ReturnedCitations)
	}
	if testCase.ExpectRefusal {
		result.CitationCorrect = result.ReturnedCitations == 0
		result.RefusalCorrect = result.CitationCorrect && isRefusal(response.Answer)
		result.Grounded = result.RefusalCorrect
		return result
	}
	result.CitationCorrect = result.ReturnedCitations > 0 && result.CorrectCitations == result.ReturnedCitations
	result.RefusalCorrect = !isRefusal(response.Answer)
	result.Grounded = strings.TrimSpace(response.Answer) != "" && result.CitationCorrect
	return result
}

func validCitation(citation port.Citation) bool {
	if citation.ArticleID <= 0 || strings.TrimSpace(citation.DocumentID) == "" || strings.TrimSpace(citation.Title) == "" {
		return false
	}
	value := strings.TrimSpace(citation.URL)
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return false
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func isRefusal(answer string) bool {
	answer = strings.TrimSpace(answer)
	return answer == InsufficientEvidenceAnswer || strings.Contains(answer, "没有足够资料") || strings.Contains(answer, "资料不足")
}
