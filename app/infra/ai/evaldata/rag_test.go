package evaldata

import "testing"

func TestLoadRAGCasesValidatesPositiveAndRefusalSamples(t *testing.T) {
	cases, err := LoadRAGCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 5 || cases[0].ExpectedArticleIDs[0] != 101 || !cases[len(cases)-1].ExpectRefusal {
		t.Fatalf("RAG cases = %#v", cases)
	}
}

func TestNormalizeRAGCasesRejectsMixedCitationAndRefusalExpectations(t *testing.T) {
	_, err := NormalizeRAGCases([]RAGCase{
		{ID: "positive", Query: "answer", ExpectedArticleIDs: []int{1}},
		{ID: "bad-refusal", Query: "none", ExpectRefusal: true, ExpectedArticleIDs: []int{1}},
	})
	if err == nil {
		t.Fatal("mixed citation/refusal case was accepted")
	}
}
