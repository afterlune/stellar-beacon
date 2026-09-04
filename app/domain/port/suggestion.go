package port

import "context"

type ContentSuggestionRequest struct {
	Title   string
	Content string
}

type ContentSuggestion struct {
	Category string
	Tags     []string
}

type ContentSuggestionGateway interface {
	Suggest(context.Context, ContentSuggestionRequest) (ContentSuggestion, error)
}
