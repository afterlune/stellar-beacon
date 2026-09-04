package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const (
	maxRadioScriptRunes = 1_200
	defaultRadioPhase   = port.AgentRhythmAwake
)

var radioHTMLTagPattern = regexp.MustCompile(`<[^>]*>`)

type RadioService interface {
	Current(context.Context) port.ResultVO
}

type RadioServiceDeps struct {
	Articles   port.ArticleRepository
	Profiles   port.AgentProfileRepository
	Rhythm     port.AgentRhythm
	ProfileID  string
	Enabled    bool
	TTSEnabled bool
	Now        func() time.Time
}

type MyRadioService struct {
	articles   port.ArticleRepository
	profiles   port.AgentProfileRepository
	rhythm     port.AgentRhythm
	profileID  string
	enabled    bool
	ttsEnabled bool
	now        func() time.Time
}

func NewRadioService(deps RadioServiceDeps) (*MyRadioService, error) {
	if deps.Enabled && deps.Articles == nil {
		return nil, apperrors.Invalid("radio.service.dependencies", "article repository is required")
	}
	if deps.Enabled && deps.Profiles == nil {
		return nil, apperrors.Invalid("radio.service.dependencies", "agent profile repository is required")
	}
	if deps.Enabled && deps.Rhythm == nil {
		return nil, apperrors.Invalid("radio.service.dependencies", "agent rhythm is required")
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &MyRadioService{
		articles:   deps.Articles,
		profiles:   deps.Profiles,
		rhythm:     deps.Rhythm,
		profileID:  strings.TrimSpace(deps.ProfileID),
		enabled:    deps.Enabled,
		ttsEnabled: deps.TTSEnabled,
		now:        deps.Now,
	}, nil
}

func NewDisabledRadioService() *MyRadioService {
	return &MyRadioService{}
}

func (s *MyRadioService) Current(ctx context.Context) port.ResultVO {
	if s == nil || !s.enabled || s.articles == nil || s.profiles == nil || s.rhythm == nil {
		return port.ResultFailWithMessage("电台暂未公开")
	}
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	snapshot := s.rhythm.Snapshot(now)
	phase := snapshot.Phase
	if phase == "" {
		phase = defaultRadioPhase
	}
	profile, err := s.profiles.Get(ctx, s.profileID)
	if err != nil {
		return port.ResultFromError(err)
	}
	article, articleErr := s.articles.GetLastArticle(ctx)
	if articleErr != nil && apperrors.KindOf(articleErr) != apperrors.KindNotFound {
		return port.ResultFromError(articleErr)
	}

	opening := cleanRadioText(profile.Opening)
	phasePrompt := cleanRadioText(profile.RhythmPrompts[phase])
	if phasePrompt == "" {
		phasePrompt = cleanRadioText(profile.RhythmPrompts[defaultRadioPhase])
	}
	articleTitle := cleanRadioText(article.ArticleTitle)
	articleSummary := cleanRadioText(article.ArticleContent)
	parts := make([]string, 0, 4)
	if opening != "" {
		parts = append(parts, opening)
	}
	if phasePrompt != "" {
		parts = append(parts, phasePrompt)
	}
	if articleTitle != "" {
		parts = append(parts, "今天的文章是《"+articleTitle+"》。")
		if articleSummary != "" {
			parts = append(parts, articleSummary)
		}
	} else {
		parts = append(parts, "今天还没有新的公开文章，先安静地听一会儿风声吧。")
	}
	script := truncateRadioText(strings.Join(parts, " "), maxRadioScriptRunes)
	if script == "" {
		script = "电台暂时没有节目。"
	}

	episodeID := radioEpisodeID(now, profile, phase, article.Id)
	episode := port.RadioEpisodeDTO{
		ID:              episodeID,
		Title:           "Benetnasch · " + radioPhaseLabel(phase),
		Script:          script,
		Phase:           string(phase),
		TTS:             s.ttsEnabled,
		SourceArticleID: article.Id,
		PublishedAt:     now.UTC(),
	}
	return port.ResultOkWithData(port.RadioPageDTO{Records: []port.RadioEpisodeDTO{episode}, Count: 1})
}

func radioEpisodeID(now time.Time, profile port.AgentProfile, phase port.AgentRhythmPhase, articleID int) string {
	value := strings.Join([]string{
		now.Format("2006-01-02"),
		profile.ID,
		profile.PromptVersion,
		string(phase),
		strconv.Itoa(articleID),
	}, "|")
	digest := sha256.Sum256([]byte(value))
	return "radio-" + hex.EncodeToString(digest[:8])
}

func cleanRadioText(value string) string {
	value = radioHTMLTagPattern.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	value = strings.ReplaceAll(value, "```", " ")
	value = strings.ReplaceAll(value, "#", " ")
	value = strings.ReplaceAll(value, "*", " ")
	value = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func truncateRadioText(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return strings.TrimSpace(string(runes[:max-1])) + "…"
}

func radioPhaseLabel(phase port.AgentRhythmPhase) string {
	switch phase {
	case port.AgentRhythmDusk:
		return "黄昏节目"
	case port.AgentRhythmNight:
		return "深夜节目"
	default:
		return "清醒节目"
	}
}

var _ RadioService = (*MyRadioService)(nil)
