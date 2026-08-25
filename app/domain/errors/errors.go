package errors

import (
	"errors"
	"fmt"
)

// Kind classifies an error at the application boundary without exposing
// infrastructure details to callers.
type Kind string

const (
	KindValidation   Kind = "validation"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindUnavailable  Kind = "unavailable"
	KindInternal     Kind = "internal"
)

// Error carries a stable classification and an operation name while keeping
// the original error available for structured logging and errors.Is/As.
type Error struct {
	Kind Kind
	Op   string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Op == "" {
		return string(e.Kind)
	}
	if e.Err == nil {
		return e.Op + ": " + string(e.Kind)
	}
	return e.Op + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && e != nil && other != nil && e.Kind == other.Kind
}

func New(kind Kind, op string, err error) error {
	if err == nil {
		err = errors.New(string(kind))
	}
	return &Error{Kind: kind, Op: op, Err: err}
}

func Wrap(kind Kind, op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Op: op, Err: err}
}

func KindOf(err error) Kind {
	var typed *Error
	if errors.As(err, &typed) && typed != nil {
		return typed.Kind
	}
	return KindInternal
}

func IsKind(err error, kind Kind) bool {
	return KindOf(err) == kind
}

func Op(err error) string {
	var typed *Error
	if errors.As(err, &typed) && typed != nil {
		return typed.Op
	}
	return ""
}

func Invalid(op, message string) error {
	return New(KindValidation, op, fmt.Errorf("%s", message))
}

func NotFound(op string) error {
	return New(KindNotFound, op, errors.New("not found"))
}

func Conflict(op, message string) error {
	return New(KindConflict, op, fmt.Errorf("%s", message))
}

func Unavailable(op string, err error) error {
	if err == nil {
		return New(KindUnavailable, op, errors.New("service unavailable"))
	}
	return Wrap(KindUnavailable, op, err)
}
