package repository

import (
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestNormalizeAgentTaskRunAppliesDefaultsAndUTC(t *testing.T) {
	created := time.Date(2026, 8, 29, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	run, err := normalizeAgentTaskRun(port.AgentTaskRun{
		RequestID: "request-1",
		SessionID: "session-1",
		Goal:      "整理文章",
		CreatedAt: created,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == "" || run.Status != port.AgentTaskQueued || run.UpdatedAt.IsZero() || run.WakeAt.IsZero() {
		t.Fatalf("normalized run = %+v", run)
	}
	if run.CreatedAt.Location() != time.UTC || run.UpdatedAt.Location() != time.UTC || run.WakeAt.Location() != time.UTC {
		t.Fatalf("normalized run times are not UTC: %+v", run)
	}
}

func TestNormalizeAgentTaskRunRejectsInvalidState(t *testing.T) {
	tests := []port.AgentTaskRun{
		{Goal: "goal", Status: "unknown"},
		{Goal: "goal", PlanRevision: -1},
		{Goal: ""},
	}
	for index, input := range tests {
		if _, err := normalizeAgentTaskRun(input); err == nil {
			t.Fatalf("case %d: invalid task state was accepted", index)
		}
	}
}

func TestNormalizeAgentPlanStepEncodesDependencies(t *testing.T) {
	step, err := normalizeAgentPlanStep(port.AgentPlanStep{
		RunID:     "run-1",
		StepID:    "step-1",
		Revision:  1,
		Ordinal:   0,
		Action:    "read_article",
		DependsOn: []string{"step-0", "step-0"},
	}, "run-1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if step.DependsOn != `["step-0","step-0"]` || step.Status != port.AgentStepPending {
		t.Fatalf("normalized step = %+v", step)
	}
	if _, err := normalizeAgentPlanStep(port.AgentPlanStep{
		RunID: "other", StepID: "step-1", Revision: 1, Action: "read_article",
	}, "run-1", 1); err == nil {
		t.Fatal("step from another run was accepted")
	}
}

func TestNormalizeAgentQuestionDeduplicatesOptions(t *testing.T) {
	question, options, err := normalizeAgentQuestion(port.AgentQuestion{
		RunID:   "run-1",
		StepID:  "step-1",
		Prompt:  "选择发布范围",
		Options: []string{"公开", "公开", "  私密  "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if question.ID == "" || question.Status != port.AgentQuestionPending || options != `["公开","私密"]` {
		t.Fatalf("question=%+v options=%s", question, options)
	}
}

func TestNormalizeAgentEffectRequiresStableIdentity(t *testing.T) {
	effect, err := normalizeAgentEffect(port.AgentEffect{
		EffectKey: "run-1:step-1:publish",
		RunID:     "run-1",
		StepID:    "step-1",
		Tool:      "publish_comment",
	})
	if err != nil {
		t.Fatal(err)
	}
	if effect.CreatedAt.IsZero() {
		t.Fatal("effect timestamp was not defaulted")
	}
	for index, input := range []port.AgentEffect{
		{RunID: "run-1", StepID: "step-1", Tool: "publish_comment"},
		{EffectKey: "key", RunID: "run-1", StepID: "step-1"},
	} {
		if _, err := normalizeAgentEffect(input); err == nil {
			t.Fatalf("case %d: invalid effect identity was accepted", index)
		}
	}
}
