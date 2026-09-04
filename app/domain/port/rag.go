package port

import "context"

// RAGRequest is the provider-neutral input for one grounded answer. The
// orchestration layer owns prompt construction; callers provide only the
// user query and optional retrieval controls.
type RAGRequest struct {
	Query  string
	Mode   SearchMode
	Filter KnowledgeFilter
}

// RAGResponse contains the generated answer and the deterministic provenance
// selected by the retriever. Citations are derived from retrieved documents,
// never from model-produced text.
type RAGResponse struct {
	Answer    string
	Citations []Citation
}

// RAGGateway is the application boundary for grounded generation. Eino and
// provider SDK types must remain behind the infrastructure implementation.
type RAGGateway interface {
	Generate(context.Context, RAGRequest) (RAGResponse, error)
}
