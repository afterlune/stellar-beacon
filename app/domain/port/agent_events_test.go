package port

import (
	"testing"
	"time"
)

func TestCanTransitionAgentTaskMatchesDurableLifecycle(t *testing.T) {
	allowed := [][2]AgentTaskStatus{
		{AgentTaskQueued, AgentTaskPlanning},
		{AgentTaskPlanning, AgentTaskRunning},
		{AgentTaskRunning, AgentTaskWaitingQuestion},
		{AgentTaskWaitingQuestion, AgentTaskRunning},
		{AgentTaskRunning, AgentTaskWaitingApproval},
		{AgentTaskWaitingApproval, AgentTaskRunning},
		{AgentTaskRunning, AgentTaskCompleted},
		{AgentTaskRunning, AgentTaskFailed},
		{AgentTaskQueued, AgentTaskCancelled},
	}
	for _, pair := range allowed {
		if !CanTransitionAgentTask(pair[0], pair[1]) {
			t.Errorf("transition %q -> %q rejected", pair[0], pair[1])
		}
	}

	denied := [][2]AgentTaskStatus{
		{AgentTaskCompleted, AgentTaskRunning},
		{AgentTaskFailed, AgentTaskQueued},
		{AgentTaskCancelled, AgentTaskRunning},
		{AgentTaskQueued, AgentTaskCompleted},
		{AgentTaskPlanning, AgentTaskWaitingApproval},
	}
	for _, pair := range denied {
		if CanTransitionAgentTask(pair[0], pair[1]) {
			t.Errorf("transition %q -> %q unexpectedly accepted", pair[0], pair[1])
		}
	}
}

func TestNewEventEnvelopeCopiesPayloadAndStartsUnpersisted(t *testing.T) {
	payload := []byte(`{"type":"delta"}`)
	event, err := NewEventEnvelope("event-1", "session-1", "turn-1", "assistant.delta", payload)
	if err != nil {
		t.Fatalf("NewEventEnvelope() error = %v", err)
	}
	payload[0] = 'X'
	if event.Seq != -1 || string(event.Payload) != `{"type":"delta"}` {
		t.Fatalf("event = %#v, want copied unpersisted envelope", event)
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestEventEnvelopeRejectsInvalidReplayIdentity(t *testing.T) {
	event := EventEnvelope{
		EventID:       "event-1",
		SessionID:     "session-1",
		TurnID:        "turn-1",
		Seq:           -1,
		Type:          "assistant.delta",
		SchemaVersion: 1,
		OccurredAt:    time.Now(),
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	event.Seq = -2
	if err := event.Validate(); err == nil {
		t.Fatal("event with invalid sequence accepted")
	}
}
