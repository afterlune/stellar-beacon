package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"xorm.io/xorm"
)

// MyAgentProfileRepository stores the administrator-controlled persona. It
// is deliberately opt-in at bootstrap because the table is introduced by an
// explicit migration and must not change existing deployments implicitly.
type MyAgentProfileRepository struct {
	engine *xorm.Engine
}

func NewAgentProfileRepository(engine *xorm.Engine) *MyAgentProfileRepository {
	return &MyAgentProfileRepository{engine: engine}
}

var _ port.AgentProfileRepository = (*MyAgentProfileRepository)(nil)

type agentProfileRow struct {
	ID              string    `xorm:"id"`
	Name            string    `xorm:"name"`
	PromptVersion   string    `xorm:"prompt_version"`
	SystemPromptRef string    `xorm:"system_prompt_ref"`
	SystemPrompt    string    `xorm:"system_prompt"`
	Opening         string    `xorm:"opening"`
	AwakePrompt     string    `xorm:"awake_prompt"`
	DuskPrompt      string    `xorm:"dusk_prompt"`
	NightPrompt     string    `xorm:"night_prompt"`
	BehaviorEnabled bool      `xorm:"behavior_enabled"`
	Enabled         bool      `xorm:"enabled"`
	UpdatedAt       time.Time `xorm:"updated_at"`
}

const agentProfileColumns = `id, name, prompt_version, system_prompt_ref,
system_prompt, opening, awake_prompt, dusk_prompt, night_prompt,
behavior_enabled, enabled, updated_at`

func (r *MyAgentProfileRepository) Get(ctx context.Context, id string) (port.AgentProfile, error) {
	if r == nil {
		return port.AgentProfile{}, apperrors.Unavailable("agent.profile.get.database", nil)
	}
	id = strings.TrimSpace(id)
	if id == "" || !validAgentProfileText(id, 128, false) {
		return port.AgentProfile{}, apperrors.Invalid("agent.profile.get", "profile id is invalid")
	}
	session, err := repoSession(r.engine, ctx, "agent.profile.get")
	if err != nil {
		return port.AgentProfile{}, err
	}
	defer session.Close()
	var row agentProfileRow
	found, err := session.SQL("SELECT "+agentProfileColumns+" FROM t_agent_profile WHERE id = ?", id).Get(&row)
	if err != nil {
		return port.AgentProfile{}, apperrors.Unavailable("agent.profile.get", err)
	}
	if !found {
		return port.AgentProfile{}, apperrors.NotFound("agent.profile.get")
	}
	profile, err := row.profile()
	if err != nil {
		return port.AgentProfile{}, apperrors.Unavailable("agent.profile.decode", err)
	}
	return profile, nil
}

func (r *MyAgentProfileRepository) Save(ctx context.Context, input port.AgentProfile) error {
	if r == nil {
		return apperrors.Unavailable("agent.profile.save.database", nil)
	}
	profile, err := normalizeAgentProfile(input)
	if err != nil {
		return apperrors.Invalid("agent.profile.save", err.Error())
	}
	return repoTx(r.engine, ctx, "agent.profile.save", func(session *xorm.Session) error {
		_, err := session.Exec(`
INSERT INTO t_agent_profile
    (id, name, prompt_version, system_prompt_ref, system_prompt, opening,
     awake_prompt, dusk_prompt, night_prompt, behavior_enabled, enabled,
     updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    prompt_version = EXCLUDED.prompt_version,
    system_prompt_ref = EXCLUDED.system_prompt_ref,
    system_prompt = EXCLUDED.system_prompt,
    opening = EXCLUDED.opening,
    awake_prompt = EXCLUDED.awake_prompt,
    dusk_prompt = EXCLUDED.dusk_prompt,
    night_prompt = EXCLUDED.night_prompt,
    behavior_enabled = EXCLUDED.behavior_enabled,
    enabled = EXCLUDED.enabled,
    updated_at = EXCLUDED.updated_at`,
			profile.ID,
			profile.Name,
			profile.PromptVersion,
			profile.SystemPromptRef,
			profile.SystemPrompt,
			profile.Opening,
			profile.RhythmPrompts[port.AgentRhythmAwake],
			profile.RhythmPrompts[port.AgentRhythmDusk],
			profile.RhythmPrompts[port.AgentRhythmNight],
			profile.BehaviorEnabled,
			profile.Enabled,
			profile.UpdatedAt,
		)
		return err
	})
}

func (r agentProfileRow) profile() (port.AgentProfile, error) {
	profile := port.AgentProfile{
		ID:              strings.TrimSpace(r.ID),
		Name:            strings.TrimSpace(r.Name),
		PromptVersion:   strings.TrimSpace(r.PromptVersion),
		SystemPromptRef: strings.TrimSpace(r.SystemPromptRef),
		SystemPrompt:    strings.TrimSpace(r.SystemPrompt),
		Opening:         strings.TrimSpace(r.Opening),
		RhythmPrompts: map[port.AgentRhythmPhase]string{
			port.AgentRhythmAwake: strings.TrimSpace(r.AwakePrompt),
			port.AgentRhythmDusk:  strings.TrimSpace(r.DuskPrompt),
			port.AgentRhythmNight: strings.TrimSpace(r.NightPrompt),
		},
		BehaviorEnabled: r.BehaviorEnabled,
		Enabled:         r.Enabled,
		UpdatedAt:       r.UpdatedAt,
	}
	if err := validateAgentProfileRecord(profile); err != nil {
		return port.AgentProfile{}, err
	}
	return profile, nil
}

func normalizeAgentProfile(profile port.AgentProfile) (port.AgentProfile, error) {
	profile.ID = strings.TrimSpace(profile.ID)
	profile.Name = strings.TrimSpace(profile.Name)
	profile.PromptVersion = strings.TrimSpace(profile.PromptVersion)
	profile.SystemPromptRef = strings.TrimSpace(profile.SystemPromptRef)
	profile.SystemPrompt = strings.TrimSpace(profile.SystemPrompt)
	profile.Opening = strings.TrimSpace(profile.Opening)
	profile.RhythmPrompts = map[port.AgentRhythmPhase]string{
		port.AgentRhythmAwake: strings.TrimSpace(profile.RhythmPrompts[port.AgentRhythmAwake]),
		port.AgentRhythmDusk:  strings.TrimSpace(profile.RhythmPrompts[port.AgentRhythmDusk]),
		port.AgentRhythmNight: strings.TrimSpace(profile.RhythmPrompts[port.AgentRhythmNight]),
	}
	if profile.SystemPromptRef == "" {
		profile.SystemPromptRef = "database:agent_profile.system_prompt"
	}
	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = time.Now().UTC()
	} else {
		profile.UpdatedAt = profile.UpdatedAt.UTC()
	}
	if err := validateAgentProfileRecord(profile); err != nil {
		return port.AgentProfile{}, err
	}
	return profile, nil
}

func validateAgentProfileRecord(profile port.AgentProfile) error {
	if !validAgentProfileText(profile.ID, 128, true) ||
		!validAgentProfileText(profile.Name, 64, true) ||
		!validAgentProfileText(profile.PromptVersion, 128, true) ||
		!validAgentProfileText(profile.SystemPromptRef, 255, true) ||
		!validAgentProfileText(profile.SystemPrompt, 8000, true) ||
		!validAgentProfileText(profile.Opening, 1200, false) {
		return apperrors.Invalid("agent.profile.validate", "profile fields are invalid")
	}
	for _, phase := range []port.AgentRhythmPhase{port.AgentRhythmAwake, port.AgentRhythmDusk, port.AgentRhythmNight} {
		if !validAgentProfileText(profile.RhythmPrompts[phase], 1200, false) {
			return apperrors.Invalid("agent.profile.validate", "rhythm prompt is invalid")
		}
	}
	return nil
}

func validAgentProfileText(value string, maxRunes int, required bool) bool {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes || (required && value == "") {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}
