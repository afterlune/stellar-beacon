package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.AgentActivityRepository = (*MyAgentActivityRepository)(nil)

type MyAgentActivityRepository struct {
	engine *xorm.Engine
}

func NewAgentActivityRepository(engine *xorm.Engine) *MyAgentActivityRepository {
	return &MyAgentActivityRepository{engine: engine}
}

type agentActivityRow struct {
	ID          string    `xorm:"id"`
	Kind        string    `xorm:"kind"`
	Emotion     string    `xorm:"emotion"`
	ContentType string    `xorm:"content_type"`
	ContentID   string    `xorm:"content_id"`
	LifeStage   string    `xorm:"life_stage"`
	Metadata    string    `xorm:"metadata"`
	OccurredAt  time.Time `xorm:"occurred_at"`
	CreatedAt   time.Time `xorm:"created_at"`
}

func (r *MyAgentActivityRepository) Append(ctx context.Context, input port.AgentActivity) error {
	activity, metadata, err := normalizeAgentActivity(input)
	if err != nil {
		return apperrors.Invalid("agent_activity.append", err.Error())
	}
	return repoTx(r.engine, ctx, "agent_activity.append", func(session *xorm.Session) error {
		_, err := session.Exec(`
INSERT INTO t_agent_activity
    (id, kind, emotion, content_type, content_id, life_stage, metadata, occurred_at)
VALUES (?, ?, ?, ?, ?, ?, ?::jsonb, ?)
ON CONFLICT (id) DO NOTHING`,
			activity.ID,
			activity.Kind,
			activity.Emotion,
			activity.ContentType,
			activity.ContentID,
			string(activity.LifeStage),
			metadata,
			activity.OccurredAt,
		)
		return err
	})
}

func (r *MyAgentActivityRepository) List(ctx context.Context, filter port.AgentActivityFilter) ([]port.AgentActivity, error) {
	filter, err := normalizeAgentActivityFilter(filter)
	if err != nil {
		return nil, apperrors.Invalid("agent_activity.list", err.Error())
	}
	session, err := repoSession(r.engine, ctx, "agent_activity.list")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	query := `SELECT id, kind, emotion, content_type, content_id, life_stage,
COALESCE(metadata::text, '{}') AS metadata, occurred_at, created_at
FROM t_agent_activity WHERE 1 = 1`
	args := make([]any, 0, 5)
	if filter.Kind != "" {
		query += " AND kind = ?"
		args = append(args, filter.Kind)
	}
	if filter.LifeStage != "" {
		query += " AND life_stage = ?"
		args = append(args, string(filter.LifeStage))
	}
	if !filter.From.IsZero() {
		query += " AND occurred_at >= ?"
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		query += " AND occurred_at < ?"
		args = append(args, filter.To)
	}
	query += " ORDER BY occurred_at DESC, id DESC LIMIT ?"
	args = append(args, filter.Limit)
	var rows []agentActivityRow
	if err := session.SQL(query, args...).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("agent_activity.list", err)
	}
	activities := make([]port.AgentActivity, 0, len(rows))
	for _, row := range rows {
		metadata := make(map[string]string)
		if strings.TrimSpace(row.Metadata) != "" {
			if err := json.Unmarshal([]byte(row.Metadata), &metadata); err != nil {
				return nil, apperrors.Unavailable("agent_activity.metadata", err)
			}
		}
		activities = append(activities, port.AgentActivity{
			ID:          row.ID,
			Kind:        row.Kind,
			Emotion:     row.Emotion,
			ContentType: row.ContentType,
			ContentID:   row.ContentID,
			LifeStage:   port.LifeStage(row.LifeStage),
			Metadata:    metadata,
			OccurredAt:  row.OccurredAt.UTC(),
		})
	}
	return activities, nil
}

func normalizeAgentActivity(input port.AgentActivity) (port.AgentActivity, string, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	input.Kind = strings.TrimSpace(input.Kind)
	input.Emotion = normalizeStoredActivityEmotion(input.Emotion)
	input.ContentType = strings.TrimSpace(input.ContentType)
	input.ContentID = strings.TrimSpace(input.ContentID)
	if input.LifeStage == "" {
		input.LifeStage = port.LifeStageGrowing
	}
	switch input.LifeStage {
	case port.LifeStageNewborn, port.LifeStageGrowing, port.LifeStageSettled, port.LifeStageForgotten:
	default:
		return port.AgentActivity{}, "", stderrors.New("activity life stage is invalid")
	}
	if len(input.ID) > 64 || input.Kind == "" || len(input.Kind) > 64 || input.Emotion == "" || len(input.ContentType) > 64 || len(input.ContentID) > 128 {
		return port.AgentActivity{}, "", stderrors.New("activity identity or metadata is invalid")
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	input.OccurredAt = input.OccurredAt.UTC()
	metadata, err := json.Marshal(input.Metadata)
	if err != nil {
		return port.AgentActivity{}, "", err
	}
	if len(metadata) > 16*1024 {
		return port.AgentActivity{}, "", stderrors.New("activity metadata is too large")
	}
	return input, string(metadata), nil
}

func normalizeStoredActivityEmotion(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "toxic", "毒舌":
		return "toxic"
	case "gentle", "温柔":
		return "gentle"
	case "melancholic", "melancholy", "忧郁":
		return "melancholic"
	case "", "neutral", "中性":
		return "neutral"
	default:
		return ""
	}
}

func normalizeAgentActivityFilter(filter port.AgentActivityFilter) (port.AgentActivityFilter, error) {
	filter.Kind = strings.TrimSpace(filter.Kind)
	if len(filter.Kind) > 64 {
		return port.AgentActivityFilter{}, stderrors.New("activity kind is too long")
	}
	if filter.LifeStage != "" {
		switch filter.LifeStage {
		case port.LifeStageNewborn, port.LifeStageGrowing, port.LifeStageSettled, port.LifeStageForgotten:
		default:
			return port.AgentActivityFilter{}, stderrors.New("activity life stage is invalid")
		}
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}
	if !filter.From.IsZero() {
		filter.From = filter.From.UTC()
	}
	if !filter.To.IsZero() {
		filter.To = filter.To.UTC()
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		return port.AgentActivityFilter{}, stderrors.New("activity time range is invalid")
	}
	return filter, nil
}
