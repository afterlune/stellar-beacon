package service

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	defaultAgentMaxInputRunes   = 4000
	defaultAgentMaxAnswerRunes  = 16000
	defaultAgentMaxToolCalls    = 4
	defaultAgentMaxConcurrent   = 2
	defaultAgentMaxOutputTokens = 1200
	maxAgentMaxOutputTokens     = 4096
)

// AgentChatLimits are application policy limits, deliberately separate from
// model-provider limits. A malformed or sensitive request is rejected before
// it can consume a provider slot or a daily quota.
type AgentChatLimits struct {
	GuestDailyTurns   int
	AdminDailyTurns   int
	MaxConcurrent     int
	MaxInputRunes     int
	MaxAnswerRunes    int
	MaxToolCalls      int
	MaxOutputTokens   int
	SensitivePatterns []string
}

func DefaultAgentChatLimits() AgentChatLimits {
	return AgentChatLimits{
		GuestDailyTurns: 20,
		AdminDailyTurns: 200,
		MaxConcurrent:   defaultAgentMaxConcurrent,
		MaxInputRunes:   defaultAgentMaxInputRunes,
		MaxAnswerRunes:  defaultAgentMaxAnswerRunes,
		MaxToolCalls:    defaultAgentMaxToolCalls,
		MaxOutputTokens: defaultAgentMaxOutputTokens,
		SensitivePatterns: []string{
			"-----begin",
			"private key",
			"akia",
			"ltai",
			"sk-",
		},
	}
}

func (l AgentChatLimits) normalize() AgentChatLimits {
	defaults := DefaultAgentChatLimits()
	if l.GuestDailyTurns <= 0 {
		l.GuestDailyTurns = defaults.GuestDailyTurns
	}
	if l.AdminDailyTurns <= 0 {
		l.AdminDailyTurns = defaults.AdminDailyTurns
	}
	if l.MaxConcurrent <= 0 {
		l.MaxConcurrent = defaults.MaxConcurrent
	}
	if l.MaxInputRunes <= 0 {
		l.MaxInputRunes = defaults.MaxInputRunes
	}
	if l.MaxAnswerRunes <= 0 {
		l.MaxAnswerRunes = defaults.MaxAnswerRunes
	}
	if l.MaxToolCalls <= 0 {
		l.MaxToolCalls = defaults.MaxToolCalls
	}
	if l.MaxOutputTokens <= 0 {
		l.MaxOutputTokens = defaults.MaxOutputTokens
	}
	if l.MaxOutputTokens > maxAgentMaxOutputTokens {
		l.MaxOutputTokens = maxAgentMaxOutputTokens
	}
	patterns := append([]string(nil), defaults.SensitivePatterns...)
	for _, pattern := range l.SensitivePatterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" || containsString(patterns, pattern) {
			continue
		}
		patterns = append(patterns, pattern)
	}
	l.SensitivePatterns = patterns
	return l
}

func validateAgentMessage(message string, limits AgentChatLimits) error {
	if !utf8.ValidString(message) {
		return errors.New("message contains invalid UTF-8")
	}
	if strings.TrimSpace(message) == "" {
		return errors.New("message is required")
	}
	if utf8.RuneCountInString(message) > limits.MaxInputRunes {
		return errors.New("message is too long")
	}
	for _, r := range message {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return errors.New("message contains unsupported control characters")
		}
	}
	lower := strings.ToLower(message)
	for _, pattern := range limits.SensitivePatterns {
		if pattern != "" && strings.Contains(lower, pattern) {
			return errors.New("message contains sensitive credential material")
		}
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
