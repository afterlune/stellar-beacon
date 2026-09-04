package service

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"strings"
	"unicode"
	"unicode/utf8"
)

type AgentProfileService interface {
	Get(c port.Request) port.ResultVO
	Update(c port.Request) port.ResultVO
}

type MyAgentProfileService struct {
	profiles  port.AgentProfileRepository
	profileID string
}

func NewAgentProfileService(profiles port.AgentProfileRepository, profileID string) *MyAgentProfileService {
	return &MyAgentProfileService{profiles: profiles, profileID: strings.TrimSpace(profileID)}
}

func (s *MyAgentProfileService) Get(c port.Request) port.ResultVO {
	profile, err := s.get(c.Context())
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(agentProfileDTO(profile))
}

func (s *MyAgentProfileService) Update(c port.Request) port.ResultVO {
	var request port.AgentProfileUpdateVO
	if err := c.BindJSON(&request); err != nil {
		return port.ResultFromError(errors.Invalid("agent.profile.request", "request body is invalid"))
	}
	profile, err := s.get(c.Context())
	if err != nil {
		return port.ResultFromError(err)
	}
	if err := applyAgentProfileUpdate(&profile, request); err != nil {
		return port.ResultFromError(err)
	}
	if err := s.profiles.Save(c.Context(), profile); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(agentProfileDTO(profile))
}

func (s *MyAgentProfileService) get(ctx context.Context) (port.AgentProfile, error) {
	if s == nil || s.profiles == nil {
		return port.AgentProfile{}, errors.Unavailable("agent.profile.service", nil)
	}
	profileID := s.profileID
	if profileID == "" {
		profileID = port.DefaultAgentProfileID
	}
	return s.profiles.Get(ctx, profileID)
}

func applyAgentProfileUpdate(profile *port.AgentProfile, request port.AgentProfileUpdateVO) error {
	if profile == nil {
		return errors.Invalid("agent.profile.update", "profile is required")
	}
	if request.Name != nil {
		profile.Name = strings.TrimSpace(*request.Name)
	}
	if request.PromptVersion != nil {
		profile.PromptVersion = strings.TrimSpace(*request.PromptVersion)
	}
	if request.SystemPrompt != nil {
		profile.SystemPrompt = strings.TrimSpace(*request.SystemPrompt)
	}
	if request.Opening != nil {
		profile.Opening = strings.TrimSpace(*request.Opening)
	}
	if request.AwakePrompt != nil {
		setRhythmPrompt(profile, port.AgentRhythmAwake, *request.AwakePrompt)
	}
	if request.DuskPrompt != nil {
		setRhythmPrompt(profile, port.AgentRhythmDusk, *request.DuskPrompt)
	}
	if request.NightPrompt != nil {
		setRhythmPrompt(profile, port.AgentRhythmNight, *request.NightPrompt)
	}
	if request.Enabled != nil {
		profile.Enabled = *request.Enabled
	}
	if err := validateAgentProfile(*profile); err != nil {
		return err
	}
	return nil
}

func setRhythmPrompt(profile *port.AgentProfile, phase port.AgentRhythmPhase, value string) {
	if profile.RhythmPrompts == nil {
		profile.RhythmPrompts = make(map[port.AgentRhythmPhase]string)
	}
	profile.RhythmPrompts[phase] = strings.TrimSpace(value)
}

func validateAgentProfile(profile port.AgentProfile) error {
	if strings.TrimSpace(profile.ID) == "" || !validProfileText(profile.ID, 128, false) {
		return errors.Invalid("agent.profile.validate", "profile id is invalid")
	}
	if !validProfileText(profile.Name, 64, true) || !validProfileText(profile.PromptVersion, 128, true) ||
		!validProfileText(profile.SystemPrompt, 8000, true) || !validProfileText(profile.Opening, 1200, false) {
		return errors.Invalid("agent.profile.validate", "profile text is invalid")
	}
	for _, phase := range []port.AgentRhythmPhase{port.AgentRhythmAwake, port.AgentRhythmDusk, port.AgentRhythmNight} {
		if !validProfileText(profile.RhythmPrompts[phase], 1200, false) {
			return errors.Invalid("agent.profile.validate", "rhythm prompt is invalid")
		}
	}
	return nil
}

func validProfileText(value string, maxRunes int, required bool) bool {
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

func agentProfileDTO(profile port.AgentProfile) port.AgentProfileDTO {
	rhythm := make(map[string]string, 3)
	for _, phase := range []port.AgentRhythmPhase{port.AgentRhythmAwake, port.AgentRhythmDusk, port.AgentRhythmNight} {
		rhythm[string(phase)] = strings.TrimSpace(profile.RhythmPrompts[phase])
	}
	return port.AgentProfileDTO{
		ID:            profile.ID,
		Name:          profile.Name,
		PromptVersion: profile.PromptVersion,
		SystemPrompt:  profile.SystemPrompt,
		Opening:       profile.Opening,
		RhythmPrompts: rhythm,
		Enabled:       profile.Enabled,
		UpdatedAt:     profile.UpdatedAt,
	}
}
