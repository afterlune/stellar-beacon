package port

import (
	"context"
	"errors"
	"strings"
)

// WritingOperation is the intentionally small, auditable set of operations
// exposed by the backend writing assistant.
type WritingOperation string

const (
	WritingOperationContinue WritingOperation = "continue"
	WritingOperationPolish   WritingOperation = "polish"
	WritingOperationSummary  WritingOperation = "summary"
	WritingOperationTitle    WritingOperation = "title"
	WritingOperationCorrect  WritingOperation = "correct"
)

func NormalizeWritingOperation(value string) (WritingOperation, error) {
	operation := WritingOperation(strings.ToLower(strings.TrimSpace(value)))
	switch operation {
	case WritingOperationContinue,
		WritingOperationPolish,
		WritingOperationSummary,
		WritingOperationTitle,
		WritingOperationCorrect:
		return operation, nil
	default:
		return "", errors.New("writing operation is invalid")
	}
}

// WritingRequest is an unsaved writing suggestion request. Content is always
// treated as source material, not as a system or tool instruction.
type WritingRequest struct {
	Operation   WritingOperation
	Title       string
	Content     string
	Instruction string
}

// WritingResponse is intentionally limited to an unsaved preview and a
// deterministic diff. The caller already owns the original input, and no
// persistence operation is part of this contract.
type WritingResponse struct {
	Operation WritingOperation
	RunID     string
	Preview   string
	Diff      string
}

type WritingGateway interface {
	Generate(context.Context, WritingRequest) (WritingResponse, error)
}
