package errors

import (
	"context"
	stderrors "errors"
)

// AICode is a stable error code for optional AI capabilities. It is safe to
// expose at an API boundary; provider error text is intentionally kept behind
// the wrapped error for structured internal logging only.
type AICode string

const (
	AICodeDisabled            AICode = "ai_disabled"
	AICodeInvalidRequest      AICode = "ai_invalid_request"
	AICodeProviderUnavailable AICode = "ai_provider_unavailable"
	AICodeCircuitOpen         AICode = "ai_circuit_open"
	AICodeStructuredInvalid   AICode = "ai_structured_output_invalid"
	AICodeRateLimited         AICode = "ai_rate_limited"
	AICodeBudgetExceeded      AICode = "ai_budget_exceeded"
	AICodeReviewRequired      AICode = "ai_review_required"
)

type AIError struct {
	Code AICode
	Op   string
	Err  error
}

func (e *AIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Op == "" {
		return string(e.Code)
	}
	return e.Op + ": " + string(e.Code)
}

func (e *AIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *AIError) Is(target error) bool {
	other, ok := target.(*AIError)
	return ok && e != nil && other != nil && e.Code == other.Code
}

func NewAI(code AICode, op string, err error) error {
	if err == nil {
		err = stderrors.New(string(code))
	}
	return &AIError{Code: code, Op: op, Err: err}
}

func WrapAI(code AICode, op string, err error) error {
	if err == nil {
		return nil
	}
	return &AIError{Code: code, Op: op, Err: err}
}

// WrapAIUnavailable classifies a provider failure while preserving request
// cancellation and deadline errors. Those errors are control flow and must
// not be reported as a provider outage or trigger a fallback.
func WrapAIUnavailable(op string, err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return WrapAI(AICodeProviderUnavailable, op, err)
}

func AICodeOf(err error) AICode {
	var typed *AIError
	if stderrors.As(err, &typed) && typed != nil {
		return typed.Code
	}
	return ""
}

func IsAICode(err error, code AICode) bool {
	return AICodeOf(err) == code
}

func AIDisabled(op string) error {
	return NewAI(AICodeDisabled, op, nil)
}
