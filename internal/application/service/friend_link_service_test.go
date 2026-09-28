package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"

	"github.com/gin-gonic/gin"
)

type applyFriendLinkRepo struct {
	fakeFriendLinkRepository
	applications []entity.TFriendLink
	existing     bool
	reviewed     []int
	reviewStatus int
}

func (f *applyFriendLinkRepo) CreateApplication(_ context.Context, link entity.TFriendLink) (int, error) {
	f.applications = append(f.applications, link)
	return len(f.applications), nil
}

func (f *applyFriendLinkRepo) AddressExists(context.Context, string) (bool, error) {
	return f.existing, nil
}

func (f *applyFriendLinkRepo) Review(_ context.Context, ids []int, status int) error {
	f.reviewed = append(f.reviewed, ids...)
	f.reviewStatus = status
	return nil
}

func friendLinkContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/public/links/applications", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestFriendLinkApplicationStoresPendingSubmission(t *testing.T) {
	repo := &applyFriendLinkRepo{}
	service := NewFriendLinkService(repo)

	result := service.ApplyFriendLink(friendLinkContext(t, `{"linkName":"站点","linkAddress":"https://site.example.test","linkIntro":"介绍","email":"me@example.test"}`))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(repo.applications) != 1 {
		t.Fatalf("expected one stored application, got %d", len(repo.applications))
	}
	stored := repo.applications[0]
	if stored.Status != port.FriendLinkStatusPending || stored.ApplicantEmail != "me@example.test" {
		t.Fatalf("unexpected stored application: %+v", stored)
	}
}

func TestFriendLinkApplicationRejectsInvalidInput(t *testing.T) {
	repo := &applyFriendLinkRepo{}
	service := NewFriendLinkService(repo)

	cases := map[string]string{
		"missing name":   `{"linkAddress":"https://site.example.test"}`,
		"relative url":   `{"linkName":"站点","linkAddress":"/local"}`,
		"bad email":      `{"linkName":"站点","linkAddress":"https://site.example.test","email":"not-an-email"}`,
		"overlong intro": `{"linkName":"站点","linkAddress":"https://site.example.test","linkIntro":"` + strings.Repeat("字", 101) + `"}`,
	}
	for name, body := range cases {
		result := service.ApplyFriendLink(friendLinkContext(t, body))
		if result.Flag {
			t.Fatalf("%s must be rejected", name)
		}
	}
	if len(repo.applications) != 0 {
		t.Fatalf("invalid submissions must not be stored: %+v", repo.applications)
	}
}

// Bots fill the hidden field; the endpoint answers OK but stores nothing.
func TestFriendLinkApplicationDropsHoneypot(t *testing.T) {
	repo := &applyFriendLinkRepo{}
	service := NewFriendLinkService(repo)

	result := service.ApplyFriendLink(friendLinkContext(t, `{"linkName":"bot","linkAddress":"https://bot.example.test","website":"https://spam"}`))
	if !result.Flag {
		t.Fatalf("honeypot submissions answer OK: %+v", result)
	}
	if len(repo.applications) != 0 {
		t.Fatal("honeypot submissions must not be stored")
	}
}

func TestFriendLinkApplicationSkipsKnownAddress(t *testing.T) {
	repo := &applyFriendLinkRepo{existing: true}
	service := NewFriendLinkService(repo)

	result := service.ApplyFriendLink(friendLinkContext(t, `{"linkName":"站点","linkAddress":"https://site.example.test"}`))
	if !result.Flag {
		t.Fatalf("duplicates answer OK: %+v", result)
	}
	if len(repo.applications) != 0 {
		t.Fatal("known addresses must not be stored twice")
	}
}

func TestFriendLinkReviewOnlyAcceptsFinalStates(t *testing.T) {
	repo := &applyFriendLinkRepo{}
	service := NewFriendLinkService(repo)

	invalid := service.ReviewFriendLinks(friendLinkContext(t, `{"ids":[3],"status":0}`))
	if invalid.Flag {
		t.Fatal("pending is not a review outcome")
	}
	approved := service.ReviewFriendLinks(friendLinkContext(t, `{"ids":[3],"status":1}`))
	if !approved.Flag || repo.reviewStatus != port.FriendLinkStatusApproved || len(repo.reviewed) != 1 {
		t.Fatalf("unexpected review result: %+v %v", approved, repo.reviewed)
	}
	empty := service.ReviewFriendLinks(friendLinkContext(t, `{"ids":[],"status":2}`))
	if empty.Flag {
		t.Fatal("an empty id list must be rejected")
	}
}
