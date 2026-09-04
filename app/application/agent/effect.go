package agent

import (
	"context"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

// EffectRunner makes the retry boundary explicit for side effects. A
// completed effect is returned from durable storage and the executor is not
// called again. The executor should therefore be reserved for operations whose
// effect key is stable across retries.
type EffectRunner struct {
	repository port.AgentTaskRepository
	now        func() time.Time
}

func NewEffectRunner(repository port.AgentTaskRepository) (*EffectRunner, error) {
	if repository == nil {
		return nil, apperrors.Unavailable("agent.effect.repository", nil)
	}
	return &EffectRunner{repository: repository, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (r *EffectRunner) Run(ctx context.Context, effect port.AgentEffect, execute func(context.Context) ([]byte, error)) ([]byte, error) {
	if r == nil || r.repository == nil {
		return nil, apperrors.Unavailable("agent.effect.run", nil)
	}
	if execute == nil {
		return nil, apperrors.Invalid("agent.effect.run", "effect executor is required")
	}
	effect.EffectKey = strings.TrimSpace(effect.EffectKey)
	effect.RunID = strings.TrimSpace(effect.RunID)
	effect.StepID = strings.TrimSpace(effect.StepID)
	effect.Tool = strings.TrimSpace(effect.Tool)
	if effect.EffectKey == "" || effect.RunID == "" || effect.StepID == "" || effect.Tool == "" {
		return nil, apperrors.Invalid("agent.effect.run", "effect identity is required")
	}

	existing, found, err := r.repository.GetEffect(ctx, effect.EffectKey)
	if err != nil {
		return nil, err
	}
	if found {
		if existing.RunID != effect.RunID || existing.StepID != effect.StepID || existing.Tool != effect.Tool {
			return nil, apperrors.Conflict("agent.effect.run", "effect key belongs to another side effect")
		}
		return cloneBytes(existing.Result), nil
	}

	result, err := execute(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "agent.effect.execute", err)
	}
	effect.Result = cloneBytes(result)
	if effect.CreatedAt.IsZero() {
		effect.CreatedAt = r.now().UTC()
	} else {
		effect.CreatedAt = effect.CreatedAt.UTC()
	}
	if err := r.repository.SaveEffect(ctx, effect); err != nil {
		return nil, err
	}

	// Read back the canonical value. This makes callers converge on the
	// durable result when two retries race after both passed the initial read.
	stored, found, err := r.repository.GetEffect(ctx, effect.EffectKey)
	if err != nil {
		return nil, err
	}
	if found {
		if stored.RunID != effect.RunID || stored.StepID != effect.StepID || stored.Tool != effect.Tool {
			return nil, apperrors.Conflict("agent.effect.run", "effect key belongs to another side effect")
		}
		return cloneBytes(stored.Result), nil
	}
	return cloneBytes(result), nil
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}
