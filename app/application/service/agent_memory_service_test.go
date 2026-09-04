package service

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"

	"github.com/gin-gonic/gin"
)

type agentMemoryServiceFake struct {
	assertions     []port.AgentMemoryAssertion
	history        []port.AgentMemoryAssertionRevision
	conflicts      []port.AgentMemoryConflict
	err            error
	listFilter     port.AgentMemoryFilter
	conflictFilter port.AgentMemoryConflictFilter
	resolved       struct{ conflictID, winnerID, resolution string }
	rejected       struct{ conflictID, resolution string }
	statusID       string
	status         port.AgentMemoryAssertionStatus
	lastContext    context.Context
}

func (f *agentMemoryServiceFake) Upsert(context.Context, port.AgentMemoryAssertion) error {
	return f.err
}

func (f *agentMemoryServiceFake) List(ctx context.Context, filter port.AgentMemoryFilter) ([]port.AgentMemoryAssertion, error) {
	f.lastContext = ctx
	f.listFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.assertions, nil
}

func (f *agentMemoryServiceFake) MarkStale(context.Context, time.Time) (int, error) {
	return 0, f.err
}

func (f *agentMemoryServiceFake) SetStatus(ctx context.Context, id string, status port.AgentMemoryAssertionStatus) error {
	f.lastContext = ctx
	f.statusID = id
	f.status = status
	return f.err
}

func (f *agentMemoryServiceFake) History(ctx context.Context, _ string) ([]port.AgentMemoryAssertionRevision, error) {
	f.lastContext = ctx
	if f.err != nil {
		return nil, f.err
	}
	return f.history, nil
}

func (f *agentMemoryServiceFake) ListConflicts(ctx context.Context, filter port.AgentMemoryConflictFilter) ([]port.AgentMemoryConflict, error) {
	f.lastContext = ctx
	f.conflictFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.conflicts, nil
}

func (f *agentMemoryServiceFake) ResolveConflict(ctx context.Context, conflictID, winnerID, resolution string) error {
	f.lastContext = ctx
	f.resolved = struct{ conflictID, winnerID, resolution string }{conflictID, winnerID, resolution}
	return f.err
}

func (f *agentMemoryServiceFake) RejectConflict(ctx context.Context, conflictID, resolution string) error {
	f.lastContext = ctx
	f.rejected = struct{ conflictID, resolution string }{conflictID, resolution}
	return f.err
}

func agentMemoryServiceContext(method, path string, body any, id string) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	var reader io.Reader = http.NoBody
	if body != nil {
		payload, _ := json.Marshal(body)
		reader = bytes.NewReader(payload)
	}
	c.Request = httptest.NewRequest(method, path, reader)
	if body != nil {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	if id != "" {
		c.Params = gin.Params{{Key: "id", Value: id}}
	}
	return serviceTestRequest{ginContextForServiceTest: c}
}

func mustAgentMemoryService(t *testing.T, fake *agentMemoryServiceFake) *MyAgentMemoryService {
	t.Helper()
	service, err := NewAgentMemoryService(AgentMemoryServiceDeps{
		Assertions: fake,
		History:    fake,
		Conflicts:  fake,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestNewAgentMemoryServiceRequiresAllRepositories(t *testing.T) {
	cases := []struct {
		name string
		deps AgentMemoryServiceDeps
	}{
		{name: "assertions", deps: AgentMemoryServiceDeps{History: &agentMemoryServiceFake{}, Conflicts: &agentMemoryServiceFake{}}},
		{name: "history", deps: AgentMemoryServiceDeps{Assertions: &agentMemoryServiceFake{}, Conflicts: &agentMemoryServiceFake{}}},
		{name: "conflicts", deps: AgentMemoryServiceDeps{Assertions: &agentMemoryServiceFake{}, History: &agentMemoryServiceFake{}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewAgentMemoryService(test.deps); err == nil || !apperrors.IsKind(err, apperrors.KindUnavailable) {
				t.Fatalf("NewAgentMemoryService() error = %v, want unavailable", err)
			}
		})
	}
}

func TestAgentMemoryServiceListsAssertionsWithNormalizedFilters(t *testing.T) {
	validFrom := time.Date(2026, 8, 29, 1, 2, 3, 0, time.FixedZone("CST", 8*60*60))
	fake := &agentMemoryServiceFake{assertions: []port.AgentMemoryAssertion{{
		ID: "assertion-1", SubjectKey: "user:42", Predicate: "likes", Object: "astronomy",
		SourceType: "chat", SourceID: "run-1", Version: 2, Confidence: 0.8,
		Status: port.AgentMemoryActive, ValidFrom: validFrom, CreatedAt: validFrom, UpdatedAt: validFrom,
	}}}
	service := mustAgentMemoryService(t, fake)
	result := service.ListAssertions(agentMemoryServiceContext(http.MethodGet, "/admin/ai/memory/assertions?current=2&size=1&subjectKey=USER:42&predicate=likes&status=ACTIVE", nil, ""))
	if !result.Flag {
		t.Fatalf("ListAssertions() failed: %+v", result)
	}
	page, ok := result.Data.(model.AgentMemoryAssertionPageDTO)
	if !ok || len(page.Items) != 1 || page.Current != 2 || page.Size != 1 || !page.HasMore {
		t.Fatalf("ListAssertions() data = %#v", result.Data)
	}
	if fake.listFilter.SubjectKey != "user:42" || fake.listFilter.Status != port.AgentMemoryActive || fake.listFilter.Current != 2 || fake.listFilter.Size != 1 {
		t.Fatalf("normalized assertion filter = %+v", fake.listFilter)
	}
	if page.Items[0].ValidFrom.Location() != time.UTC || page.Items[0].Status != string(port.AgentMemoryActive) {
		t.Fatalf("assertion DTO = %+v", page.Items[0])
	}
}

func TestAgentMemoryServiceListsHistoryAndConflicts(t *testing.T) {
	fake := &agentMemoryServiceFake{
		history:   []port.AgentMemoryAssertionRevision{{AssertionID: "assertion-1", RevisionNo: 1, Object: "old", Status: port.AgentMemoryStale}},
		conflicts: []port.AgentMemoryConflict{{ID: "conflict-1", SubjectKey: "user:42", Predicate: "likes", Status: port.AgentMemoryConflictOpen, Members: []port.AgentMemoryConflictMember{{AssertionID: "assertion-1", Role: port.AgentMemoryConflictCandidate}}}},
	}
	service := mustAgentMemoryService(t, fake)

	historyResult := service.ListHistory(agentMemoryServiceContext(http.MethodGet, "/admin/ai/memory/assertions/assertion-1/history", nil, "assertion-1"))
	history, ok := historyResult.Data.(model.AgentMemoryHistoryDTO)
	if !historyResult.Flag || !ok || history.AssertionID != "assertion-1" || len(history.Items) != 1 {
		t.Fatalf("ListHistory() = %+v", historyResult)
	}

	conflictResult := service.ListConflicts(agentMemoryServiceContext(http.MethodGet, "/admin/ai/memory/conflicts?subjectKey=user:42&status=open", nil, ""))
	page, ok := conflictResult.Data.(model.AgentMemoryConflictPageDTO)
	if !conflictResult.Flag || !ok || len(page.Items) != 1 || page.Items[0].Members[0].Role != string(port.AgentMemoryConflictCandidate) {
		t.Fatalf("ListConflicts() = %+v", conflictResult)
	}
	if fake.conflictFilter.SubjectKey != "user:42" || fake.conflictFilter.Status != port.AgentMemoryConflictOpen {
		t.Fatalf("normalized conflict filter = %+v", fake.conflictFilter)
	}
}

func TestAgentMemoryServiceResolvesRejectsAndRevokes(t *testing.T) {
	fake := &agentMemoryServiceFake{}
	service := mustAgentMemoryService(t, fake)

	resolve := service.ResolveConflict(agentMemoryServiceContext(http.MethodPost, "/admin/ai/memory/conflicts/conflict-1/resolve", model.AgentMemoryResolveVO{WinnerAssertionID: "assertion-1", Resolution: "verified by operator"}, "conflict-1"))
	if !resolve.Flag || fake.resolved.conflictID != "conflict-1" || fake.resolved.winnerID != "assertion-1" || fake.resolved.resolution != "verified by operator" {
		t.Fatalf("ResolveConflict() = %+v, call = %+v", resolve, fake.resolved)
	}

	reject := service.RejectConflict(agentMemoryServiceContext(http.MethodPost, "/admin/ai/memory/conflicts/conflict-1/reject", nil, "conflict-1"))
	if !reject.Flag || fake.rejected.conflictID != "conflict-1" || fake.rejected.resolution != "" {
		t.Fatalf("RejectConflict() = %+v, call = %+v", reject, fake.rejected)
	}

	revoke := service.RevokeAssertion(agentMemoryServiceContext(http.MethodDelete, "/admin/ai/memory/assertions/assertion-1", nil, "assertion-1"))
	if !revoke.Flag || fake.statusID != "assertion-1" || fake.status != port.AgentMemoryRetracted {
		t.Fatalf("RevokeAssertion() = %+v, id=%q status=%q", revoke, fake.statusID, fake.status)
	}
}

func TestAgentMemoryServiceRejectsInvalidRequests(t *testing.T) {
	fake := &agentMemoryServiceFake{}
	service := mustAgentMemoryService(t, fake)

	invalidPage := service.ListAssertions(agentMemoryServiceContext(http.MethodGet, "/admin/ai/memory/assertions?current=0", nil, ""))
	if invalidPage.Flag || invalidPage.Message != "参数格式不正确" {
		t.Fatalf("invalid page result = %+v", invalidPage)
	}

	invalidResolve := service.ResolveConflict(agentMemoryServiceContext(http.MethodPost, "/admin/ai/memory/conflicts/conflict-1/resolve", map[string]string{"winnerAssertionId": ""}, "conflict-1"))
	if invalidResolve.Flag || invalidResolve.Message != "参数格式不正确" {
		t.Fatalf("invalid resolve result = %+v", invalidResolve)
	}

	invalidSubject := service.ListConflicts(agentMemoryServiceContext(http.MethodGet, "/admin/ai/memory/conflicts?subjectKey=session:short", nil, ""))
	if invalidSubject.Flag || invalidSubject.Message != "参数格式不正确" {
		t.Fatalf("invalid subject result = %+v", invalidSubject)
	}
}

func TestAgentMemoryServiceDoesNotExposeRepositoryDetailsAndCanBeDisabled(t *testing.T) {
	fake := &agentMemoryServiceFake{err: apperrors.Unavailable("agent_memory.list", stderrors.New("password=secret"))}
	service := mustAgentMemoryService(t, fake)
	result := service.ListAssertions(agentMemoryServiceContext(http.MethodGet, "/admin/ai/memory/assertions", nil, ""))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" || result.Message == "password=secret" {
		t.Fatalf("repository error leaked or succeeded: %+v", result)
	}

	disabled := NewDisabledAgentMemoryService()
	result = disabled.ListAssertions(agentMemoryServiceContext(http.MethodGet, "/admin/ai/memory/assertions", nil, ""))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("disabled service result = %+v", result)
	}
}
