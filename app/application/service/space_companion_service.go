package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
)

const (
	spaceProtocolVersion = "1"
	spaceSourceLimit     = 100
	spaceBodyLimit       = 4_000
)

// SpaceCompanionService is the application boundary used by the internal
// Companion protocol. It exposes only public projections and the narrow
// publication allowlist; it has no admin, credential, draft or delete method.
type SpaceCompanionService interface {
	Capabilities() port.SpaceCapabilities
	Search(context.Context, port.SpaceSearchQuery) (port.SpaceSearchResult, error)
	Get(context.Context, port.SpaceContentType, string) (port.SpaceContent, error)
	Publish(context.Context, port.SpacePrincipal, port.SpacePublicationInput) (port.SpacePublicationResult, error)
}

type SpaceCompanionServiceDeps struct {
	Articles     port.ArticleRepository
	Albums       port.PhotoAlbumRepository
	Photos       port.PhotoRepository
	Dreams       port.DreamRepository
	Videos       port.VideoRepository
	Radio        RadioService
	Publications port.SpacePublicationRepository
	AgentID      string
	Enabled      bool
	Publish      bool
	Now          func() time.Time
}

type MySpaceCompanionService struct {
	articles     port.ArticleRepository
	albums       port.PhotoAlbumRepository
	photos       port.PhotoRepository
	dreams       port.DreamRepository
	videos       port.VideoRepository
	radio        RadioService
	publications port.SpacePublicationRepository
	agentID      string
	enabled      bool
	publish      bool
	now          func() time.Time
}

func NewSpaceCompanionService(deps SpaceCompanionServiceDeps) (*MySpaceCompanionService, error) {
	agentID := strings.TrimSpace(deps.AgentID)
	if agentID == "" {
		agentID = port.MoonfeiPrincipalID
	}
	if len(agentID) > 128 || strings.ContainsAny(agentID, "\r\n\x00") {
		return nil, apperrors.Invalid("space.companion.dependencies", "agent id is invalid")
	}
	if agentID != port.MoonfeiPrincipalID {
		return nil, apperrors.Invalid("space.companion.dependencies", "agent id is not the configured resident")
	}
	if deps.Publish && deps.Publications == nil {
		return nil, apperrors.Invalid("space.companion.dependencies", "publication repository is required")
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &MySpaceCompanionService{
		articles:     deps.Articles,
		albums:       deps.Albums,
		photos:       deps.Photos,
		dreams:       deps.Dreams,
		videos:       deps.Videos,
		radio:        deps.Radio,
		publications: deps.Publications,
		agentID:      agentID,
		enabled:      deps.Enabled,
		publish:      deps.Publish,
		now:          deps.Now,
	}, nil
}

func NewDisabledSpaceCompanionService() *MySpaceCompanionService {
	return &MySpaceCompanionService{}
}

func (s *MySpaceCompanionService) Capabilities() port.SpaceCapabilities {
	agentID := port.MoonfeiPrincipalID
	readEnabled := false
	publishEnabled := false
	if s != nil {
		if strings.TrimSpace(s.agentID) != "" {
			agentID = s.agentID
		}
		readEnabled = s.enabled
		publishEnabled = s.enabled && s.publish
	}
	scopes := []string{port.SpaceScopeRead}
	if publishEnabled {
		scopes = append(scopes, port.SpaceScopePublish)
	}
	return port.SpaceCapabilities{
		ProtocolVersion: spaceProtocolVersion,
		Principal: port.SpacePrincipal{
			ID:     agentID,
			Type:   port.SpacePrincipalAgent,
			Scopes: scopes,
		},
		ReadEnabled:      readEnabled,
		PublishEnabled:   publishEnabled,
		ReadableTypes:    []port.SpaceContentType{port.SpaceContentArticle, port.SpaceContentPhoto, port.SpaceContentVideo, port.SpaceContentDream, port.SpaceContentRadio, port.SpaceContentStatus},
		PublishableTypes: []port.SpaceContentType{port.SpaceContentStatus, port.SpaceContentDream, port.SpaceContentRadio},
	}
}

func (s *MySpaceCompanionService) Search(ctx context.Context, query port.SpaceSearchQuery) (port.SpaceSearchResult, error) {
	if s == nil || !s.enabled {
		return port.SpaceSearchResult{}, apperrors.Unavailable("space.companion.search", nil)
	}
	normalized, types, err := query.Normalize()
	if err != nil {
		return port.SpaceSearchResult{}, apperrors.Invalid("space.companion.search", err.Error())
	}
	items, err := s.collect(ctx, types, normalized.Query)
	if err != nil {
		return port.SpaceSearchResult{}, err
	}
	sortSpaceContent(items)
	total := len(items)
	if len(items) > normalized.Limit {
		items = items[:normalized.Limit]
	}
	return port.SpaceSearchResult{Items: items, Count: total}, nil
}

func (s *MySpaceCompanionService) Get(ctx context.Context, contentType port.SpaceContentType, id string) (port.SpaceContent, error) {
	if s == nil || !s.enabled {
		return port.SpaceContent{}, apperrors.Unavailable("space.companion.get", nil)
	}
	normalizedType, err := port.NormalizeSpaceContentType(string(contentType))
	if err != nil {
		return port.SpaceContent{}, apperrors.Invalid("space.companion.get", err.Error())
	}
	id = strings.TrimSpace(id)
	if id == "" || len(id) > port.MaxSpaceContentID || strings.ContainsAny(id, "/\\\r\n\x00") {
		return port.SpaceContent{}, apperrors.Invalid("space.companion.get", "content id is invalid")
	}
	if normalizedType == port.SpaceContentStatus || normalizedType == port.SpaceContentDream || normalizedType == port.SpaceContentRadio {
		if s.publications != nil {
			publication, publicationErr := s.publications.GetPublic(ctx, normalizedType, id)
			if publicationErr == nil {
				return spacePublicationContent(publication), nil
			}
			if apperrors.KindOf(publicationErr) != apperrors.KindNotFound {
				return port.SpaceContent{}, publicationErr
			}
		}
	}
	items, err := s.collect(ctx, []port.SpaceContentType{normalizedType}, "")
	if err != nil {
		return port.SpaceContent{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return port.SpaceContent{}, apperrors.NotFound("space.companion.get")
}

func (s *MySpaceCompanionService) Publish(ctx context.Context, principal port.SpacePrincipal, input port.SpacePublicationInput) (port.SpacePublicationResult, error) {
	if s == nil || !s.enabled || !s.publish || s.publications == nil {
		return port.SpacePublicationResult{}, apperrors.Forbidden("space.companion.publish")
	}
	if err := principal.ValidateFor(port.SpaceScopePublish); err != nil {
		return port.SpacePublicationResult{}, apperrors.Unauthorized("space.companion.publish")
	}
	if principal.ID != s.agentID {
		return port.SpacePublicationResult{}, apperrors.Forbidden("space.companion.publish")
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	publication, err := input.Normalize(s.agentID, now)
	if err != nil {
		return port.SpacePublicationResult{}, apperrors.Invalid("space.companion.publish", err.Error())
	}
	publication.ID = "publication-" + uuid.NewString()
	publication.RequestDigest = publicationDigest(publication)
	created, existing, err := s.publications.Create(ctx, publication)
	if err != nil {
		return port.SpacePublicationResult{}, err
	}
	return port.SpacePublicationResult{Publication: created, Existing: existing}, nil
}

func (s *MySpaceCompanionService) collect(ctx context.Context, types []port.SpaceContentType, query string) ([]port.SpaceContent, error) {
	requested := make(map[port.SpaceContentType]struct{}, len(types))
	for _, contentType := range types {
		requested[contentType] = struct{}{}
	}
	items := make([]port.SpaceContent, 0, spaceSourceLimit)
	if _, ok := requested[port.SpaceContentArticle]; ok && s.articles != nil {
		articles, _, err := s.articles.ListArticles(ctx, 1, spaceSourceLimit)
		if err != nil {
			return nil, err
		}
		for _, article := range articles {
			if article == nil || !port.IsPublicArticle(article.Status, 0) {
				continue
			}
			items = append(items, port.SpaceContent{
				ID:          strconv.Itoa(article.Id),
				Type:        port.SpaceContentArticle,
				Title:       strings.TrimSpace(article.ArticleTitle),
				Body:        boundedRunes(article.ArticleContent, spaceBodyLimit),
				URL:         "/articles/" + strconv.Itoa(article.Id),
				MediaURL:    strings.TrimSpace(article.ArticleCover),
				PublishedAt: articleTime(article.UpdateTime, article.CreateTime),
				Metadata:    map[string]string{"category": strings.TrimSpace(article.CategoryName)},
			})
		}
	}
	if _, ok := requested[port.SpaceContentPhoto]; ok && s.albums != nil && s.photos != nil {
		albums, err := s.albums.ListPublic(ctx)
		if err != nil {
			return nil, err
		}
		for _, album := range albums {
			if album.IsDelete != 0 || album.Status != 1 {
				continue
			}
			photos, _, photoErr := s.photos.List(ctx, 1, spaceSourceLimit, album.Id, 0)
			if photoErr != nil {
				return nil, photoErr
			}
			for _, photo := range photos {
				if photo.IsDelete != 0 || strings.TrimSpace(photo.PhotoSrc) == "" {
					continue
				}
				items = append(items, port.SpaceContent{
					ID:          strconv.Itoa(photo.Id),
					Type:        port.SpaceContentPhoto,
					Title:       firstNonEmpty(photo.PhotoName, album.AlbumName),
					Body:        boundedRunes(photo.PhotoDesc, spaceBodyLimit),
					URL:         "/albums/" + strconv.Itoa(album.Id) + "/photos",
					MediaURL:    strings.TrimSpace(photo.PhotoSrc),
					PublishedAt: articleTime(photo.UpdateTime, photo.CreateTime),
					Metadata:    map[string]string{"album": strings.TrimSpace(album.AlbumName)},
				})
			}
		}
	}
	if _, ok := requested[port.SpaceContentVideo]; ok && s.videos != nil {
		videos, _, err := s.videos.ListPublic(ctx, 1, spaceSourceLimit)
		if err != nil {
			return nil, err
		}
		for _, video := range videos {
			if video.Deleted || !video.Published {
				continue
			}
			items = append(items, port.SpaceContent{
				ID:          video.ID,
				Type:        port.SpaceContentVideo,
				Title:       video.Title,
				Body:        boundedRunes(video.Description, spaceBodyLimit),
				URL:         "/videos/" + video.ID,
				MediaURL:    firstNonEmpty(video.EmbedURL, video.URL),
				PublishedAt: articleTime(video.UpdatedAt, video.CreatedAt),
				Metadata:    map[string]string{"source": string(video.Source), "mimeType": video.MIMEType},
			})
		}
	}
	if _, ok := requested[port.SpaceContentDream]; ok && s.dreams != nil {
		dreams, _, err := s.dreams.ListPublic(ctx, 1, spaceSourceLimit)
		if err != nil {
			return nil, err
		}
		for _, dream := range dreams {
			if dream.Status != port.DreamApproved {
				continue
			}
			items = append(items, port.SpaceContent{
				ID:          dream.ID,
				Type:        port.SpaceContentDream,
				Title:       dream.Title,
				Body:        boundedRunes(dream.Content, spaceBodyLimit),
				MediaURL:    dream.ImageURL,
				PublishedAt: articleTime(dream.UpdatedAt, dream.CreatedAt),
				Metadata:    map[string]string{"imageStatus": string(dream.ImageStatus)},
			})
		}
	}
	if _, ok := requested[port.SpaceContentRadio]; ok && s.radio != nil {
		result := s.radio.Current(ctx)
		if result.Flag {
			if page, ok := result.Data.(port.RadioPageDTO); ok {
				for _, episode := range page.Records {
					items = append(items, port.SpaceContent{
						ID:          episode.ID,
						Type:        port.SpaceContentRadio,
						Title:       episode.Title,
						Body:        boundedRunes(episode.Script, spaceBodyLimit),
						PublishedAt: episode.PublishedAt.UTC(),
						Metadata:    map[string]string{"phase": episode.Phase},
					})
				}
			}
		}
	}
	if s.publications != nil {
		for _, publicationType := range []port.SpaceContentType{port.SpaceContentStatus, port.SpaceContentDream, port.SpaceContentRadio} {
			if _, ok := requested[publicationType]; !ok {
				continue
			}
			// Query each publication type independently so a busy status stream
			// cannot consume the bounded page and hide newer dreams or radio
			// entries requested by the caller.
			publications, _, err := s.publications.ListPublic(ctx, port.SpacePublicationQuery{
				Type:  string(publicationType),
				Query: query,
				Limit: spaceSourceLimit,
			})
			if err != nil {
				return nil, err
			}
			for _, publication := range publications {
				items = append(items, spacePublicationContent(publication))
			}
		}
	}
	if query != "" {
		needle := strings.ToLower(strings.TrimSpace(query))
		filtered := items[:0]
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Title), needle) || strings.Contains(strings.ToLower(item.Body), needle) || strings.Contains(strings.ToLower(item.MediaURL), needle) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return dedupeSpaceContent(items), nil
}

func spacePublicationContent(publication port.SpacePublication) port.SpaceContent {
	return port.SpaceContent{
		ID:          publication.ID,
		Type:        publication.Type,
		Title:       publication.Title,
		Body:        boundedRunes(publication.Body, spaceBodyLimit),
		MediaURL:    publication.MediaURL,
		PublishedAt: publication.PublishedAt.UTC(),
		Metadata: map[string]string{
			"agentId":     publication.AgentID,
			"sourceRunId": publication.SourceRunID,
		},
	}
}

func publicationDigest(publication port.SpacePublication) string {
	value, _ := json.Marshal(struct {
		AgentID         string                `json:"agentId"`
		Type            port.SpaceContentType `json:"type"`
		Title           string                `json:"title"`
		Body            string                `json:"body"`
		MediaURL        string                `json:"mediaUrl"`
		SourceSessionID string                `json:"sourceSessionId"`
		SourceRunID     string                `json:"sourceRunId"`
		IdempotencyKey  string                `json:"idempotencyKey"`
	}{publication.AgentID, publication.Type, publication.Title, publication.Body, publication.MediaURL, publication.SourceSessionID, publication.SourceRunID, publication.IdempotencyKey})
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func sortSpaceContent(items []port.SpaceContent) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].PublishedAt.Equal(items[j].PublishedAt) {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		}
		if items[i].Type != items[j].Type {
			return items[i].Type < items[j].Type
		}
		return items[i].ID < items[j].ID
	})
}

func dedupeSpaceContent(items []port.SpaceContent) []port.SpaceContent {
	seen := make(map[string]struct{}, len(items))
	result := items[:0]
	for _, item := range items {
		key := string(item.Type) + ":" + item.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func boundedRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func articleTime(primary, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary.UTC()
	}
	return fallback.UTC()
}

var _ SpaceCompanionService = (*MySpaceCompanionService)(nil)
