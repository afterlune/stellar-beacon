package port

import (
	"context"
	"errors"
	"strings"
	"time"
)

// VideoSource identifies where a public video is served from. The source is
// persisted so clients never have to infer whether a URL is an object-storage
// asset or an external frame.
type VideoSource string

const (
	VideoSourceLocal    VideoSource = "local"
	VideoSourceExternal VideoSource = "external"
)

const (
	MaxVideoBytes            int64 = 500 * 1024 * 1024
	MaxVideoIDLength               = 128
	MaxVideoTitleRunes             = 160
	MaxVideoDescriptionRunes       = 2_000
	MaxVideoMIMETypeLength         = 128
	MaxVideoPageSize               = 100
)

// Video is the application-facing representation of a moderated video
// record. It deliberately contains no upload identity or storage credentials.
type Video struct {
	ID          string
	Title       string
	Description string
	Source      VideoSource
	URL         string
	EmbedURL    string
	MIMEType    string
	SizeBytes   int64
	Published   bool
	Deleted     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NormalizeVideoSource(value string) (VideoSource, error) {
	source := VideoSource(strings.ToLower(strings.TrimSpace(value)))
	switch source {
	case VideoSourceLocal, VideoSourceExternal:
		return source, nil
	default:
		return "", errors.New("video source is invalid")
	}
}

// ValidateVideo checks persistence invariants. URL origin policy and local
// file signatures are application concerns because they depend on runtime
// configuration and multipart bytes.
func ValidateVideo(video Video) error {
	video.ID = strings.TrimSpace(video.ID)
	if video.ID == "" || len(video.ID) > MaxVideoIDLength || strings.ContainsAny(video.ID, "\r\n\x00") {
		return errors.New("video id is invalid")
	}
	video.Title = strings.TrimSpace(video.Title)
	if video.Title == "" || len([]rune(video.Title)) > MaxVideoTitleRunes || strings.ContainsAny(video.Title, "\x00\r\n") {
		return errors.New("video title is invalid")
	}
	if len([]rune(video.Description)) > MaxVideoDescriptionRunes || strings.ContainsAny(video.Description, "\x00") {
		return errors.New("video description is invalid")
	}
	source, err := NormalizeVideoSource(string(video.Source))
	if err != nil {
		return err
	}
	if strings.TrimSpace(video.URL) == "" {
		return errors.New("video URL is required")
	}
	if source == VideoSourceExternal && strings.TrimSpace(video.EmbedURL) == "" {
		return errors.New("external video embed URL is required")
	}
	if source == VideoSourceLocal && video.SizeBytes <= 0 {
		return errors.New("local video size is invalid")
	}
	if video.SizeBytes < 0 || video.SizeBytes > MaxVideoBytes {
		return errors.New("video size is invalid")
	}
	if len(video.MIMEType) > MaxVideoMIMETypeLength || strings.ContainsAny(video.MIMEType, "\r\n\x00") {
		return errors.New("video MIME type is invalid")
	}
	return nil
}

// VideoRepository is the persistence boundary for public video metadata.
// Upload bytes are handled by ObjectStorage before the metadata is created.
type VideoRepository interface {
	Create(context.Context, Video) error
	ListPublic(context.Context, int, int) ([]Video, int, error)
	Delete(context.Context, string) error
}
