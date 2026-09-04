package writing

import (
	"fmt"
	"strings"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

// OperationLimit bounds every user-controlled and model-produced field of a
// writing operation. Limits are measured in Unicode code points so Chinese
// content is not unfairly penalized by UTF-8 byte length.
type OperationLimit struct {
	MaxTitleRunes       int
	MaxContentRunes     int
	MaxInstructionRunes int
	MaxOutputRunes      int
}

const (
	defaultWritingTitleLimit       = 256
	defaultWritingInstructionLimit = 2_000
	defaultContinueContentLimit    = 12_000
	defaultContinueOutputLimit     = 6_000
	defaultPolishContentLimit      = 20_000
	defaultPolishOutputLimit       = 20_000
	defaultSummaryContentLimit     = 30_000
	defaultSummaryOutputLimit      = 4_000
	defaultTitleContentLimit       = 30_000
	defaultTitleOutputLimit        = 256
	defaultCorrectContentLimit     = 20_000
	defaultCorrectOutputLimit      = 20_000
	defaultStructuredInputLimit    = 30_000
)

// DefaultOperationLimits returns a fresh copy so callers can safely adjust a
// single operation without mutating package defaults or another assistant.
func DefaultOperationLimits() map[port.WritingOperation]OperationLimit {
	return map[port.WritingOperation]OperationLimit{
		port.WritingOperationContinue: {
			MaxTitleRunes:       defaultWritingTitleLimit,
			MaxContentRunes:     defaultContinueContentLimit,
			MaxInstructionRunes: defaultWritingInstructionLimit,
			MaxOutputRunes:      defaultContinueOutputLimit,
		},
		port.WritingOperationPolish: {
			MaxTitleRunes:       defaultWritingTitleLimit,
			MaxContentRunes:     defaultPolishContentLimit,
			MaxInstructionRunes: defaultWritingInstructionLimit,
			MaxOutputRunes:      defaultPolishOutputLimit,
		},
		port.WritingOperationSummary: {
			MaxTitleRunes:       defaultWritingTitleLimit,
			MaxContentRunes:     defaultSummaryContentLimit,
			MaxInstructionRunes: defaultWritingInstructionLimit,
			MaxOutputRunes:      defaultSummaryOutputLimit,
		},
		port.WritingOperationTitle: {
			MaxTitleRunes:       defaultWritingTitleLimit,
			MaxContentRunes:     defaultTitleContentLimit,
			MaxInstructionRunes: defaultWritingInstructionLimit,
			MaxOutputRunes:      defaultTitleOutputLimit,
		},
		port.WritingOperationCorrect: {
			MaxTitleRunes:       defaultWritingTitleLimit,
			MaxContentRunes:     defaultCorrectContentLimit,
			MaxInstructionRunes: defaultWritingInstructionLimit,
			MaxOutputRunes:      defaultCorrectOutputLimit,
		},
	}
}

func normalizeOperationLimits(overrides map[port.WritingOperation]OperationLimit) (map[port.WritingOperation]OperationLimit, error) {
	limits := DefaultOperationLimits()
	for operation, override := range overrides {
		defaults, ok := limits[operation]
		if !ok {
			return nil, apperrors.Invalid("agent.writing.limits", "unknown writing operation")
		}
		if override.MaxTitleRunes < 0 || override.MaxContentRunes < 0 || override.MaxInstructionRunes < 0 || override.MaxOutputRunes < 0 {
			return nil, apperrors.Invalid("agent.writing.limits", "writing limits cannot be negative")
		}
		if override.MaxTitleRunes > 0 {
			defaults.MaxTitleRunes = override.MaxTitleRunes
		}
		if override.MaxContentRunes > 0 {
			defaults.MaxContentRunes = override.MaxContentRunes
		}
		if override.MaxInstructionRunes > 0 {
			defaults.MaxInstructionRunes = override.MaxInstructionRunes
		}
		if override.MaxOutputRunes > 0 {
			defaults.MaxOutputRunes = override.MaxOutputRunes
		}
		if defaults.MaxTitleRunes < 1 || defaults.MaxContentRunes < 1 || defaults.MaxInstructionRunes < 1 || defaults.MaxOutputRunes < 1 {
			return nil, apperrors.Invalid("agent.writing.limits", "writing limits must be positive")
		}
		limits[operation] = defaults
	}
	return limits, nil
}

func validateWritingInput(operation port.WritingOperation, request port.WritingRequest, limits OperationLimit) error {
	if limit := len([]rune(request.Title)); limit > limits.MaxTitleRunes {
		return apperrors.Invalid("agent.writing.title", fmt.Sprintf("writing title exceeds %d characters", limits.MaxTitleRunes))
	}
	if limit := len([]rune(request.Content)); limit > limits.MaxContentRunes {
		return apperrors.Invalid("agent.writing.content", fmt.Sprintf("%s content exceeds %d characters", operation, limits.MaxContentRunes))
	}
	if limit := len([]rune(request.Instruction)); limit > limits.MaxInstructionRunes {
		return apperrors.Invalid("agent.writing.instruction", fmt.Sprintf("writing instruction exceeds %d characters", limits.MaxInstructionRunes))
	}
	return nil
}

func validateGeneratedWritingOutput(output string, limit int) error {
	if len([]rune(output)) > limit {
		return apperrors.NewAI(apperrors.AICodeStructuredInvalid, "agent.writing.output_limit", fmt.Errorf("writing output exceeds %d characters", limit))
	}
	return nil
}

func normalizeStructuredInput(value string, field string, maxRunes int) (string, error) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > maxRunes {
		return "", apperrors.Invalid("agent.writing."+field, fmt.Sprintf("writing %s exceeds %d characters", field, maxRunes))
	}
	return value, nil
}
