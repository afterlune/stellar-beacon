package service

import (
	stderrors "errors"
	"io"
	"strconv"
	"strings"

	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

// AgentMemoryService is an authenticated review surface. It is deliberately
// separate from AgentChatService: public chat has no memory repository or
// memory write capability.
type AgentMemoryService interface {
	ListAssertions(port.Request) port.ResultVO
	ListHistory(port.Request) port.ResultVO
	ListConflicts(port.Request) port.ResultVO
	ResolveConflict(port.Request) port.ResultVO
	RejectConflict(port.Request) port.ResultVO
	RevokeAssertion(port.Request) port.ResultVO
}

type AgentMemoryServiceDeps struct {
	Assertions port.AgentMemoryAssertionRepository
	History    port.AgentMemoryHistoryRepository
	Conflicts  port.AgentMemoryConflictRepository
}

type MyAgentMemoryService struct {
	assertions port.AgentMemoryAssertionRepository
	history    port.AgentMemoryHistoryRepository
	conflicts  port.AgentMemoryConflictRepository
}

var _ AgentMemoryService = (*MyAgentMemoryService)(nil)

func NewAgentMemoryService(deps AgentMemoryServiceDeps) (*MyAgentMemoryService, error) {
	if deps.Assertions == nil {
		return nil, errors.Unavailable("service.agent_memory.assertions", nil)
	}
	if deps.History == nil {
		return nil, errors.Unavailable("service.agent_memory.history", nil)
	}
	if deps.Conflicts == nil {
		return nil, errors.Unavailable("service.agent_memory.conflicts", nil)
	}
	return &MyAgentMemoryService{
		assertions: deps.Assertions,
		history:    deps.History,
		conflicts:  deps.Conflicts,
	}, nil
}

type disabledAgentMemoryService struct{}

var _ AgentMemoryService = (*disabledAgentMemoryService)(nil)

func NewDisabledAgentMemoryService() AgentMemoryService {
	return &disabledAgentMemoryService{}
}

func (s *disabledAgentMemoryService) ListAssertions(port.Request) port.ResultVO {
	return port.ResultFromError(errors.Unavailable("agent.memory", nil))
}

func (s *disabledAgentMemoryService) ListHistory(port.Request) port.ResultVO {
	return port.ResultFromError(errors.Unavailable("agent.memory", nil))
}

func (s *disabledAgentMemoryService) ListConflicts(port.Request) port.ResultVO {
	return port.ResultFromError(errors.Unavailable("agent.memory", nil))
}

func (s *disabledAgentMemoryService) ResolveConflict(port.Request) port.ResultVO {
	return port.ResultFromError(errors.Unavailable("agent.memory", nil))
}

func (s *disabledAgentMemoryService) RejectConflict(port.Request) port.ResultVO {
	return port.ResultFromError(errors.Unavailable("agent.memory", nil))
}

func (s *disabledAgentMemoryService) RevokeAssertion(port.Request) port.ResultVO {
	return port.ResultFromError(errors.Unavailable("agent.memory", nil))
}

func (s *MyAgentMemoryService) ListAssertions(c port.Request) port.ResultVO {
	if s == nil || s.assertions == nil {
		return port.ResultFromError(errors.Unavailable("agent.memory", nil))
	}
	filter, err := memoryAssertionFilter(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	assertions, err := s.assertions.List(c.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	items := make([]port.AgentMemoryAssertionDTO, 0, len(assertions))
	for _, assertion := range assertions {
		items = append(items, port.NewAgentMemoryAssertionDTO(assertion))
	}
	return port.ResultOkWithData(port.AgentMemoryAssertionPageDTO{
		Items: items, Current: filter.Current, Size: filter.Size, HasMore: len(items) == filter.Size,
	})
}

func (s *MyAgentMemoryService) ListHistory(c port.Request) port.ResultVO {
	if s == nil || s.history == nil {
		return port.ResultFromError(errors.Unavailable("agent.memory.history", nil))
	}
	assertionID, err := memoryIDParam(c, "agent.memory.history")
	if err != nil {
		return port.ResultFromError(err)
	}
	history, err := s.history.History(c.Context(), assertionID)
	if err != nil {
		return port.ResultFromError(err)
	}
	items := make([]port.AgentMemoryAssertionRevisionDTO, 0, len(history))
	for _, revision := range history {
		items = append(items, port.NewAgentMemoryAssertionRevisionDTO(revision))
	}
	return port.ResultOkWithData(port.AgentMemoryHistoryDTO{AssertionID: assertionID, Items: items})
}

func (s *MyAgentMemoryService) ListConflicts(c port.Request) port.ResultVO {
	if s == nil || s.conflicts == nil {
		return port.ResultFromError(errors.Unavailable("agent.memory.conflicts", nil))
	}
	filter, err := memoryConflictFilter(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	conflicts, err := s.conflicts.ListConflicts(c.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	items := make([]port.AgentMemoryConflictDTO, 0, len(conflicts))
	for _, conflict := range conflicts {
		items = append(items, port.NewAgentMemoryConflictDTO(conflict))
	}
	return port.ResultOkWithData(port.AgentMemoryConflictPageDTO{
		Items: items, Current: filter.Current, Size: filter.Size, HasMore: len(items) == filter.Size,
	})
}

func (s *MyAgentMemoryService) ResolveConflict(c port.Request) port.ResultVO {
	if s == nil || s.conflicts == nil {
		return port.ResultFromError(errors.Unavailable("agent.memory.conflicts", nil))
	}
	conflictID, err := memoryIDParam(c, "agent.memory.resolve")
	if err != nil {
		return port.ResultFromError(err)
	}
	var request port.AgentMemoryResolveVO
	if err := c.BindJSON(&request); err != nil {
		return port.ResultFromError(errors.Invalid("agent.memory.resolve.request", "request body is invalid"))
	}
	winnerID, err := normalizeMemoryServiceID(request.WinnerAssertionID, "winner assertion id")
	if err != nil {
		return port.ResultFromError(errors.Invalid("agent.memory.resolve.request", err.Error()))
	}
	resolution, err := normalizeMemoryResolution(request.Resolution)
	if err != nil {
		return port.ResultFromError(errors.Invalid("agent.memory.resolve.request", err.Error()))
	}
	if err := s.conflicts.ResolveConflict(c.Context(), conflictID, winnerID, resolution); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(map[string]string{"id": conflictID, "status": string(port.AgentMemoryConflictResolved)})
}

func (s *MyAgentMemoryService) RejectConflict(c port.Request) port.ResultVO {
	if s == nil || s.conflicts == nil {
		return port.ResultFromError(errors.Unavailable("agent.memory.conflicts", nil))
	}
	conflictID, err := memoryIDParam(c, "agent.memory.reject")
	if err != nil {
		return port.ResultFromError(err)
	}
	var request port.AgentMemoryRejectVO
	if c.HTTPRequest() != nil && c.HTTPRequest().Body != nil {
		if err := c.BindJSON(&request); err != nil {
			if err != io.EOF {
				return port.ResultFromError(errors.Invalid("agent.memory.reject.request", "request body is invalid"))
			}
		}
	}
	resolution, err := normalizeMemoryResolution(request.Resolution)
	if err != nil {
		return port.ResultFromError(errors.Invalid("agent.memory.reject.request", err.Error()))
	}
	if err := s.conflicts.RejectConflict(c.Context(), conflictID, resolution); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(map[string]string{"id": conflictID, "status": string(port.AgentMemoryConflictRejected)})
}

func (s *MyAgentMemoryService) RevokeAssertion(c port.Request) port.ResultVO {
	if s == nil || s.assertions == nil {
		return port.ResultFromError(errors.Unavailable("agent.memory", nil))
	}
	assertionID, err := memoryIDParam(c, "agent.memory.revoke")
	if err != nil {
		return port.ResultFromError(err)
	}
	if err := s.assertions.SetStatus(c.Context(), assertionID, port.AgentMemoryRetracted); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(map[string]string{"id": assertionID, "status": string(port.AgentMemoryRetracted)})
}

func memoryAssertionFilter(c port.Request) (port.AgentMemoryFilter, error) {
	current, size, err := memoryPageQuery(c, "agent.memory.page")
	if err != nil {
		return port.AgentMemoryFilter{}, err
	}
	filter := port.AgentMemoryFilter{Current: current, Size: size, Predicate: strings.TrimSpace(c.Query("predicate"))}
	if subject := strings.TrimSpace(c.Query("subjectKey")); subject != "" {
		filter.SubjectKey, err = port.NormalizeAgentMemorySubject(subject)
		if err != nil {
			return port.AgentMemoryFilter{}, errors.Invalid("agent.memory.subject", err.Error())
		}
	}
	if filter.Predicate != "" && (len(filter.Predicate) > 128 || strings.ContainsAny(filter.Predicate, "\r\n\x00")) {
		return port.AgentMemoryFilter{}, errors.Invalid("agent.memory.predicate", "predicate is invalid")
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		normalized, normalizeErr := port.NormalizeAgentMemoryStatus(status)
		if normalizeErr != nil {
			return port.AgentMemoryFilter{}, errors.Invalid("agent.memory.status", normalizeErr.Error())
		}
		filter.Status = normalized
	}
	return filter, nil
}

func memoryConflictFilter(c port.Request) (port.AgentMemoryConflictFilter, error) {
	current, size, err := memoryPageQuery(c, "agent.memory.conflict.page")
	if err != nil {
		return port.AgentMemoryConflictFilter{}, err
	}
	filter := port.AgentMemoryConflictFilter{Current: current, Size: size, Predicate: strings.TrimSpace(c.Query("predicate"))}
	if subject := strings.TrimSpace(c.Query("subjectKey")); subject != "" {
		filter.SubjectKey, err = port.NormalizeAgentMemorySubject(subject)
		if err != nil {
			return port.AgentMemoryConflictFilter{}, errors.Invalid("agent.memory.subject", err.Error())
		}
	}
	if filter.Predicate != "" && (len(filter.Predicate) > 128 || strings.ContainsAny(filter.Predicate, "\r\n\x00")) {
		return port.AgentMemoryConflictFilter{}, errors.Invalid("agent.memory.predicate", "predicate is invalid")
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		normalized, normalizeErr := port.NormalizeAgentMemoryConflictStatus(status)
		if normalizeErr != nil {
			return port.AgentMemoryConflictFilter{}, errors.Invalid("agent.memory.conflict.status", normalizeErr.Error())
		}
		filter.Status = normalized
	}
	return filter, nil
}

func memoryPageQuery(c port.Request, operation string) (int, int, error) {
	if c == nil {
		return 0, 0, errors.Invalid(operation, "request context is required")
	}
	current, size := 1, 20
	var err error
	if value := strings.TrimSpace(c.Query("current")); value != "" {
		current, err = strconv.Atoi(value)
		if err != nil || current < 1 {
			return 0, 0, errors.Invalid(operation, "current must be a positive integer")
		}
	}
	if value := strings.TrimSpace(c.Query("size")); value != "" {
		size, err = strconv.Atoi(value)
		if err != nil || size < 1 || size > 100 {
			return 0, 0, errors.Invalid(operation, "size must be between 1 and 100")
		}
	}
	return current, size, nil
}

func memoryIDParam(c port.Request, operation string) (string, error) {
	if c == nil {
		return "", errors.Invalid(operation, "request context is required")
	}
	return normalizeMemoryServiceID(c.Param("id"), "memory id")
}

func normalizeMemoryServiceID(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 || strings.ContainsAny(value, "\r\n\x00") {
		return "", stderrors.New(label + " is invalid")
	}
	return value, nil
}

func normalizeMemoryResolution(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > 256 || strings.ContainsAny(value, "\r\n\x00") {
		return "", stderrors.New("conflict resolution is invalid")
	}
	return value, nil
}
