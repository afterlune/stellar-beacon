package port

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"
)

// SpacePrincipal is the identity established by the internal service-token
// boundary. It is never populated from a publication request body.
type SpacePrincipal struct {
	ID     string   `json:"id"`
	Type   string   `json:"type"`
	Scopes []string `json:"scopes,omitempty"`
}

const (
	SpacePrincipalAgent      = "agent"
	SpaceScopeRead           = "space:read"
	SpaceScopePublish        = "space:publish"
	MoonfeiPrincipalID       = "moonfei"
	SpacePrincipalContextKey = "spacePrincipal"
)

func (p SpacePrincipal) HasScope(scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	for _, value := range p.Scopes {
		if strings.TrimSpace(value) == scope {
			return true
		}
	}
	return false
}

func (p SpacePrincipal) Validate() error {
	if strings.TrimSpace(p.ID) == "" || len(p.ID) > 128 || strings.ContainsAny(p.ID, "\r\n\x00") {
		return errors.New("space principal is invalid")
	}
	if strings.TrimSpace(p.Type) != SpacePrincipalAgent {
		return errors.New("space principal type is invalid")
	}
	return nil
}

func (p SpacePrincipal) ValidateFor(scope string) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if !p.HasScope(scope) {
		return errors.New("space principal scope is missing")
	}
	return nil
}

// SpaceContentType is the deliberately small public knowledge vocabulary.
// Private capsules, drafts, logs, users, credentials and configuration have
// no value in this enum and therefore cannot be requested through this port.
type SpaceContentType string

const (
	SpaceContentArticle SpaceContentType = "article"
	SpaceContentPhoto   SpaceContentType = "photo"
	SpaceContentVideo   SpaceContentType = "video"
	SpaceContentDream   SpaceContentType = "dream"
	SpaceContentRadio   SpaceContentType = "radio"
	SpaceContentStatus  SpaceContentType = "status"
)

var publicSpaceContentTypes = map[SpaceContentType]struct{}{
	SpaceContentArticle: {},
	SpaceContentPhoto:   {},
	SpaceContentVideo:   {},
	SpaceContentDream:   {},
	SpaceContentRadio:   {},
	SpaceContentStatus:  {},
}

func NormalizeSpaceContentType(value string) (SpaceContentType, error) {
	typeValue := SpaceContentType(strings.ToLower(strings.TrimSpace(value)))
	if _, ok := publicSpaceContentTypes[typeValue]; !ok {
		return "", errors.New("space content type is not public")
	}
	return typeValue, nil
}

func NormalizeSpaceContentTypes(values []string) ([]SpaceContentType, error) {
	if len(values) == 0 {
		return []SpaceContentType{
			SpaceContentArticle,
			SpaceContentPhoto,
			SpaceContentVideo,
			SpaceContentDream,
			SpaceContentRadio,
			SpaceContentStatus,
		}, nil
	}
	result := make([]SpaceContentType, 0, len(values))
	seen := make(map[SpaceContentType]struct{}, len(values))
	for _, value := range values {
		contentType, err := NormalizeSpaceContentType(value)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[contentType]; ok {
			continue
		}
		seen[contentType] = struct{}{}
		result = append(result, contentType)
	}
	return result, nil
}

const (
	MaxSpaceSearchQuery      = 256
	MaxSpaceSearchLimit      = 50
	MaxSpaceContentID        = 160
	MaxSpaceTitleRunes       = 160
	MaxSpaceBodyRunes        = 20_000
	MaxSpaceSourceIDRunes    = 160
	MaxSpaceIdempotencyRunes = 255
)

type SpaceSearchQuery struct {
	Query string
	Types []string
	Limit int
}

func (q SpaceSearchQuery) Normalize() (SpaceSearchQuery, []SpaceContentType, error) {
	q.Query = strings.TrimSpace(q.Query)
	if q.Query == "" || len([]rune(q.Query)) > MaxSpaceSearchQuery || strings.ContainsAny(q.Query, "\r\n\x00") {
		return SpaceSearchQuery{}, nil, errors.New("space search query is invalid")
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 || q.Limit > MaxSpaceSearchLimit {
		return SpaceSearchQuery{}, nil, errors.New("space search limit is invalid")
	}
	types, err := NormalizeSpaceContentTypes(q.Types)
	if err != nil {
		return SpaceSearchQuery{}, nil, err
	}
	return q, types, nil
}

// SpaceContent is a bounded public projection. Body is already truncated by
// the application service and never contains a password, private identity or
// provider metadata.
type SpaceContent struct {
	ID          string            `json:"id"`
	Type        SpaceContentType  `json:"type"`
	Title       string            `json:"title"`
	Body        string            `json:"body,omitempty"`
	URL         string            `json:"url,omitempty"`
	MediaURL    string            `json:"mediaUrl,omitempty"`
	PublishedAt time.Time         `json:"publishedAt"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type SpaceSearchResult struct {
	Items []SpaceContent `json:"items"`
	Count int            `json:"count"`
}

type SpaceCapabilities struct {
	ProtocolVersion  string             `json:"protocolVersion"`
	Principal        SpacePrincipal     `json:"principal"`
	ReadEnabled      bool               `json:"readEnabled"`
	PublishEnabled   bool               `json:"publishEnabled"`
	ReadableTypes    []SpaceContentType `json:"readableTypes"`
	PublishableTypes []SpaceContentType `json:"publishableTypes"`
}

// SpacePrincipalRecord is the database-backed identity used by the internal
// protocol. Enabled is deliberately kept outside SpacePrincipal so an
// operational disable flag is never exposed as a client capability.
type SpacePrincipalRecord struct {
	Principal   SpacePrincipal
	DisplayName string
	Enabled     bool
}

// SpacePrincipalRepository is the only application-facing lookup for the
// machine principal. The middleware uses it to make a missing or disabled
// migration row fail closed instead of trusting configuration alone.
type SpacePrincipalRepository interface {
	Get(context.Context, string) (SpacePrincipalRecord, error)
}

type SpaceContentReader interface {
	Search(context.Context, SpaceSearchQuery) (SpaceSearchResult, error)
	Get(context.Context, SpaceContentType, string) (SpaceContent, error)
}

// SpacePublicationInput is the only write shape exposed to Companion. The
// service derives ActorID from SpacePrincipal and derives the request digest;
// neither can be supplied by an untrusted caller.
type SpacePublicationInput struct {
	Type            string `json:"type"`
	Title           string `json:"title"`
	Body            string `json:"body"`
	MediaURL        string `json:"mediaUrl,omitempty"`
	SourceSessionID string `json:"sourceSessionId"`
	SourceRunID     string `json:"sourceRunId"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type SpacePublication struct {
	ID              string           `json:"id"`
	AgentID         string           `json:"agentId"`
	Type            SpaceContentType `json:"type"`
	Title           string           `json:"title"`
	Body            string           `json:"body"`
	MediaURL        string           `json:"mediaUrl,omitempty"`
	SourceSessionID string           `json:"sourceSessionId"`
	SourceRunID     string           `json:"sourceRunId"`
	IdempotencyKey  string           `json:"idempotencyKey"`
	RequestDigest   string           `json:"-"`
	PublishedAt     time.Time        `json:"publishedAt"`
	CreatedAt       time.Time        `json:"createdAt"`
}

type SpacePublicationResult struct {
	Publication SpacePublication `json:"publication"`
	Existing    bool             `json:"existing"`
}

func (p SpacePublicationInput) Normalize(agentID string, now time.Time) (SpacePublication, error) {
	contentType, err := NormalizeSpacePublicationType(p.Type)
	if err != nil {
		return SpacePublication{}, err
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" || len(agentID) > 128 || strings.ContainsAny(agentID, "\r\n\x00") {
		return SpacePublication{}, errors.New("space publication agent is invalid")
	}
	title := strings.TrimSpace(p.Title)
	if title == "" || len([]rune(title)) > MaxSpaceTitleRunes || strings.ContainsAny(title, "\r\n\x00") {
		return SpacePublication{}, errors.New("space publication title is invalid")
	}
	body := strings.TrimSpace(p.Body)
	if body == "" || len([]rune(body)) > MaxSpaceBodyRunes || strings.ContainsAny(body, "\x00") {
		return SpacePublication{}, errors.New("space publication body is invalid")
	}
	mediaURL := strings.TrimSpace(p.MediaURL)
	if mediaURL != "" {
		parsed, parseErr := url.Parse(mediaURL)
		if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
			return SpacePublication{}, errors.New("space publication media URL is invalid")
		}
		if len(mediaURL) > 2048 {
			return SpacePublication{}, errors.New("space publication media URL is too long")
		}
	}
	sessionID, runID, idempotencyKey := strings.TrimSpace(p.SourceSessionID), strings.TrimSpace(p.SourceRunID), strings.TrimSpace(p.IdempotencyKey)
	for _, value := range []string{sessionID, runID} {
		if value == "" || len([]rune(value)) > MaxSpaceSourceIDRunes || strings.ContainsAny(value, "\r\n\x00") {
			return SpacePublication{}, errors.New("space publication source is invalid")
		}
	}
	if idempotencyKey == "" || len([]rune(idempotencyKey)) > MaxSpaceIdempotencyRunes || strings.ContainsAny(idempotencyKey, "\r\n\x00") {
		return SpacePublication{}, errors.New("space publication idempotency key is invalid")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return SpacePublication{
		AgentID:         agentID,
		Type:            contentType,
		Title:           title,
		Body:            body,
		MediaURL:        mediaURL,
		SourceSessionID: sessionID,
		SourceRunID:     runID,
		IdempotencyKey:  idempotencyKey,
		PublishedAt:     now.UTC(),
		CreatedAt:       now.UTC(),
	}, nil
}

func NormalizeSpacePublicationType(value string) (SpaceContentType, error) {
	contentType := SpaceContentType(strings.ToLower(strings.TrimSpace(value)))
	switch contentType {
	case SpaceContentStatus, SpaceContentDream, SpaceContentRadio:
		return contentType, nil
	default:
		return "", errors.New("space publication type is not allowed")
	}
}

type SpacePublicationQuery struct {
	Type  string
	Query string
	Limit int
}

func (q SpacePublicationQuery) Normalize() (SpacePublicationQuery, error) {
	if q.Type != "" {
		contentType, err := NormalizeSpacePublicationType(q.Type)
		if err != nil {
			return SpacePublicationQuery{}, err
		}
		q.Type = string(contentType)
	}
	q.Query = strings.TrimSpace(q.Query)
	if len([]rune(q.Query)) > MaxSpaceSearchQuery || strings.ContainsAny(q.Query, "\r\n\x00") {
		return SpacePublicationQuery{}, errors.New("space publication query is invalid")
	}
	if q.Limit == 0 {
		q.Limit = MaxSpaceSearchLimit
	}
	if q.Limit < 1 || q.Limit > MaxSpaceSearchLimit {
		return SpacePublicationQuery{}, errors.New("space publication limit is invalid")
	}
	return q, nil
}

type SpacePublicationRepository interface {
	Create(context.Context, SpacePublication) (SpacePublication, bool, error)
	GetPublic(context.Context, SpaceContentType, string) (SpacePublication, error)
	ListPublic(context.Context, SpacePublicationQuery) ([]SpacePublication, int, error)
}
