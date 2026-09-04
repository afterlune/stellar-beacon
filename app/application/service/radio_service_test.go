package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
)

type radioArticleRepositoryFake struct {
	port.ArticleRepository
	article port.ArticleCard
	err     error
}

func (f *radioArticleRepositoryFake) GetLastArticle(context.Context) (port.ArticleCard, error) {
	return f.article, f.err
}

type radioProfileRepositoryFake struct {
	port.AgentProfileRepository
	profile port.AgentProfile
	err     error
}

func (f *radioProfileRepositoryFake) Get(context.Context, string) (port.AgentProfile, error) {
	return f.profile, f.err
}

type radioRhythmFake struct {
	snapshot port.AgentRhythmSnapshot
}

func (f radioRhythmFake) Snapshot(time.Time) port.AgentRhythmSnapshot {
	return f.snapshot
}

func TestRadioServiceBuildsDeterministicPublicTextWithoutSystemPrompt(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	service, err := NewRadioService(RadioServiceDeps{
		Articles: &radioArticleRepositoryFake{article: port.ArticleCard{
			Id:             7,
			ArticleTitle:   "<b>星河</b>",
			ArticleContent: "一段 <script>不应执行</script> 文章。",
		}},
		Profiles: &radioProfileRepositoryFake{profile: port.AgentProfile{
			ID: "public-v2", PromptVersion: "v2", SystemPrompt: "private system prompt",
			Opening: "你好，<em>朋友</em>。", RhythmPrompts: map[port.AgentRhythmPhase]string{
				port.AgentRhythmAwake: "保持清醒。",
			},
		}},
		Rhythm:     radioRhythmFake{snapshot: port.AgentRhythmSnapshot{Phase: port.AgentRhythmAwake}},
		ProfileID:  "public-v2",
		Enabled:    true,
		TTSEnabled: true,
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	result := service.Current(context.Background())
	if !result.Flag {
		t.Fatalf("radio result=%+v", result)
	}
	page, ok := result.Data.(model.RadioPageDTO)
	if !ok || len(page.Records) != 1 {
		t.Fatalf("radio page=%#v", result.Data)
	}
	episode := page.Records[0]
	if !episode.TTS || episode.SourceArticleID != 7 || episode.Phase != "awake" {
		t.Fatalf("episode=%+v", episode)
	}
	if episode.Script == "" || containsAny(episode.Script, "<script>", "private system prompt", "<b>") {
		t.Fatalf("unsafe/private radio script=%q", episode.Script)
	}
	second := service.Current(context.Background()).Data.(model.RadioPageDTO).Records[0]
	if episode.ID != second.ID || episode.Script != second.Script {
		t.Fatalf("radio episode is not deterministic: first=%+v second=%+v", episode, second)
	}
}

func TestRadioServiceFailsClosedWhenDependenciesAreMissing(t *testing.T) {
	if _, err := NewRadioService(RadioServiceDeps{Enabled: true}); err == nil {
		t.Fatal("expected dependency validation error")
	}
	if result := NewDisabledRadioService().Current(context.Background()); result.Flag || result.Message != "电台暂未公开" {
		t.Fatalf("disabled result=%+v", result)
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
