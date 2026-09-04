package agent

import (
	"context"
	"errors"
	"testing"

	"benetnasch/app/domain/port"
)

type effectRepositoryFake struct {
	port.AgentTaskRepository
	effects map[string]port.AgentEffect
	gets    int
	saves   int
}

func (f *effectRepositoryFake) GetEffect(_ context.Context, key string) (port.AgentEffect, bool, error) {
	f.gets++
	effect, ok := f.effects[key]
	return effect, ok, nil
}

func (f *effectRepositoryFake) SaveEffect(_ context.Context, effect port.AgentEffect) error {
	f.saves++
	if f.effects == nil {
		f.effects = make(map[string]port.AgentEffect)
	}
	if _, exists := f.effects[effect.EffectKey]; !exists {
		f.effects[effect.EffectKey] = effect
	}
	return nil
}

func TestEffectRunnerReusesCompletedEffect(t *testing.T) {
	repository := &effectRepositoryFake{}
	runner, err := NewEffectRunner(repository)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	effect := port.AgentEffect{EffectKey: "run-1:step-1:publish", RunID: "run-1", StepID: "step-1", Tool: "publish"}
	execute := func(context.Context) ([]byte, error) {
		calls++
		return []byte("published-1"), nil
	}
	first, err := runner.Run(context.Background(), effect, execute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runner.Run(context.Background(), effect, execute)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || string(first) != "published-1" || string(second) != "published-1" || repository.saves != 1 {
		t.Fatalf("calls=%d saves=%d first=%q second=%q", calls, repository.saves, first, second)
	}
	first[0] = 'X'
	if string(repository.effects[effect.EffectKey].Result) != "published-1" {
		t.Fatal("stored effect was mutated by caller")
	}
}

func TestEffectRunnerDoesNotPersistFailedExecution(t *testing.T) {
	repository := &effectRepositoryFake{}
	runner, err := NewEffectRunner(repository)
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("provider unavailable")
	_, err = runner.Run(context.Background(), port.AgentEffect{
		EffectKey: "key", RunID: "run", StepID: "step", Tool: "tool",
	}, func(context.Context) ([]byte, error) {
		return nil, want
	})
	if err == nil || !errors.Is(err, want) {
		t.Fatalf("error=%v, want wrapped provider error", err)
	}
	if repository.saves != 0 {
		t.Fatalf("failed effect was persisted: %d", repository.saves)
	}
}

func TestEffectRunnerRejectsIdentityCollision(t *testing.T) {
	repository := &effectRepositoryFake{effects: map[string]port.AgentEffect{
		"key": {EffectKey: "key", RunID: "other-run", StepID: "step", Tool: "tool", Result: []byte("old")},
	}}
	runner, err := NewEffectRunner(repository)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	_, err = runner.Run(context.Background(), port.AgentEffect{EffectKey: "key", RunID: "run", StepID: "step", Tool: "tool"}, func(context.Context) ([]byte, error) {
		calls++
		return []byte("new"), nil
	})
	if err == nil || calls != 0 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
