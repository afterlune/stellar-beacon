package model

import (
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"testing"
)

func TestResultFromErrorDoesNotExposeInfrastructureDetails(t *testing.T) {
	result := ResultFromError(apperrors.Unavailable("article.list", assertErr("secret database detail")))
	if result.Flag {
		t.Fatal("infrastructure error must be a failed result")
	}
	if result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected public message: %q", result.Message)
	}
	if result.Message == "secret database detail" {
		t.Fatal("internal error detail leaked to response")
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }

func assertErr(message string) error { return assertionError(message) }
