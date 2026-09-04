package service

import "testing"

func TestValidateAgentMessageRejectsOversizedControlAndCredentialInput(t *testing.T) {
	limits := DefaultAgentChatLimits()
	if err := validateAgentMessage("-----BEGIN PRIVATE KEY-----", limits); err == nil {
		t.Fatal("credential material was accepted")
	}
	if err := validateAgentMessage("line\x00break", limits); err == nil {
		t.Fatal("control character was accepted")
	}
	large := make([]rune, limits.MaxInputRunes+1)
	for index := range large {
		large[index] = '字'
	}
	if err := validateAgentMessage(string(large), limits); err == nil {
		t.Fatal("oversized message was accepted")
	}
}

func TestAgentChatLimitsNormalizeKeepsSafeDefaults(t *testing.T) {
	limits := (AgentChatLimits{}).normalize()
	if limits.GuestDailyTurns != 20 || limits.AdminDailyTurns != 200 || limits.MaxConcurrent != 2 || limits.MaxToolCalls != 4 || limits.MaxOutputTokens != defaultAgentMaxOutputTokens {
		t.Fatalf("normalized limits = %#v", limits)
	}
}

func TestAgentChatLimitsNormalizeCapsProviderOutputBudget(t *testing.T) {
	limits := (AgentChatLimits{MaxOutputTokens: maxAgentMaxOutputTokens + 1}).normalize()
	if limits.MaxOutputTokens != maxAgentMaxOutputTokens {
		t.Fatalf("MaxOutputTokens = %d, want cap %d", limits.MaxOutputTokens, maxAgentMaxOutputTokens)
	}
}
