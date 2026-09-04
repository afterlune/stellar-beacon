package rag

import (
	"context"
	"strconv"
	"testing"

	"benetnasch/app/domain/port"
	"benetnasch/app/infra/ai/evaldata"
)

type evaluationGateway struct {
	responses map[string]port.RAGResponse
}

func (g evaluationGateway) Generate(_ context.Context, request port.RAGRequest) (port.RAGResponse, error) {
	return g.responses[request.Query], nil
}

func TestEvaluateRAGCalculatesGroundedCitationAndRefusalMetrics(t *testing.T) {
	cases := []evaldata.RAGCase{
		{ID: "positive", Query: "answer", ExpectedArticleIDs: []int{7}},
		{ID: "refusal", Query: "none", ExpectRefusal: true},
	}
	report, err := EvaluateRAG(context.Background(), evaluationGateway{responses: map[string]port.RAGResponse{
		"answer": {
			Answer:    "根据公开资料可以确认。",
			Citations: []port.Citation{{ArticleID: 7, DocumentID: "article-7-chunk-0", Title: "公开文章", URL: "/articles/7"}},
		},
		"none": {Answer: InsufficientEvidenceAnswer},
	}}, cases)
	if err != nil {
		t.Fatal(err)
	}
	if report.PositiveCases != 1 || report.RefusalCases != 1 || report.Groundedness != 1 || report.CitationCorrectness != 1 || report.RefusalAccuracy != 1 {
		t.Fatalf("RAG report = %#v", report)
	}
}

func TestEvaluateRAGRejectsWrongCitationAndFalseRefusal(t *testing.T) {
	cases := []evaldata.RAGCase{
		{ID: "positive", Query: "answer", ExpectedArticleIDs: []int{7}},
		{ID: "refusal", Query: "none", ExpectRefusal: true},
	}
	report, err := EvaluateRAG(context.Background(), evaluationGateway{responses: map[string]port.RAGResponse{
		"answer": {
			Answer:    "无来源的回答",
			Citations: []port.Citation{{ArticleID: 8, DocumentID: "article-8", Title: "错误来源", URL: "/articles/8"}},
		},
		"none": {Answer: "我猜答案是蓝色月球数据库。", Citations: []port.Citation{{ArticleID: 7, DocumentID: "wrong", Title: "错误", URL: "/articles/7"}}},
	}}, cases)
	if err != nil {
		t.Fatal(err)
	}
	if report.Groundedness != 0 || report.CitationCorrectness != 0 || report.RefusalAccuracy != 0 || report.Cases[0].CitationCorrect || report.Cases[1].RefusalCorrect {
		t.Fatalf("unsafe RAG report = %#v", report)
	}
}

func TestEvaluateEmbeddedRAGUsesFixedDataset(t *testing.T) {
	cases, err := evaldata.LoadRAGCases()
	if err != nil {
		t.Fatal(err)
	}
	responses := make(map[string]port.RAGResponse, len(cases))
	for _, testCase := range cases {
		if testCase.ExpectRefusal {
			responses[testCase.Query] = port.RAGResponse{Answer: InsufficientEvidenceAnswer}
			continue
		}
		citations := make([]port.Citation, 0, len(testCase.ExpectedArticleIDs))
		for _, articleID := range testCase.ExpectedArticleIDs {
			citations = append(citations, port.Citation{ArticleID: articleID, DocumentID: "article", Title: "文章", URL: "/articles/" + strconv.Itoa(articleID)})
		}
		responses[testCase.Query] = port.RAGResponse{Answer: "有公开依据。", Citations: citations}
	}
	report, err := EvaluateEmbeddedRAG(context.Background(), evaluationGateway{responses: responses})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalCases != len(cases) || report.Groundedness != 1 || report.RefusalAccuracy != 1 {
		t.Fatalf("embedded report = %#v", report)
	}
}
