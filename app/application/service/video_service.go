package service

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
)

const (
	defaultVideoPageSize = 20
	maxVideoPageSize     = port.MaxVideoPageSize
)

var videoMIMEByExtension = map[string]string{
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".mov":  "video/quicktime",
	".webm": "video/webm",
	".ogv":  "video/ogg",
}

type VideoService interface {
	List(context.Context, VideoQuery) port.ResultVO
	Upload(context.Context, VideoUploadInput) port.ResultVO
	CreateExternal(context.Context, ExternalVideoInput) port.ResultVO
	Delete(context.Context, string) port.ResultVO
	AllowedFrameOrigins() []string
}

type VideoQuery struct {
	Current string
	Size    string
}

type VideoUploadInput struct {
	File        *multipart.FileHeader
	Title       string
	Description string
}

type ExternalVideoInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type VideoServiceDeps struct {
	Repository     port.VideoRepository
	Storage        port.ObjectStorage
	AllowedOrigins []string
	Enabled        bool
	Now            func() time.Time
}

type MyVideoService struct {
	repository     port.VideoRepository
	storage        port.ObjectStorage
	allowedOrigins []string
	enabled        bool
	now            func() time.Time
}

func NewVideoService(deps VideoServiceDeps) (*MyVideoService, error) {
	if deps.Enabled && deps.Repository == nil {
		return nil, apperrors.Invalid("video.service.dependencies", "video repository is required")
	}
	if deps.Enabled && deps.Storage == nil {
		return nil, apperrors.Invalid("video.service.dependencies", "object storage is required")
	}
	origins, err := normalizeVideoOrigins(deps.AllowedOrigins)
	if err != nil {
		return nil, err
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &MyVideoService{
		repository:     deps.Repository,
		storage:        deps.Storage,
		allowedOrigins: origins,
		enabled:        deps.Enabled,
		now:            deps.Now,
	}, nil
}

func NewDisabledVideoService() *MyVideoService {
	return &MyVideoService{}
}

func (s *MyVideoService) List(ctx context.Context, query VideoQuery) port.ResultVO {
	if s == nil || !s.enabled || s.repository == nil {
		return port.ResultFailWithMessage("视频暂未公开")
	}
	current, size, err := normalizeVideoQuery(query)
	if err != nil {
		return port.ResultFromError(err)
	}
	videos, count, err := s.repository.ListPublic(ctx, current, size)
	if err != nil {
		return port.ResultFromError(err)
	}
	records := make([]port.VideoDTO, 0, len(videos))
	for _, video := range videos {
		if video.Deleted || !video.Published {
			continue
		}
		records = append(records, toVideoDTO(video))
	}
	return port.ResultOkWithData(port.VideoPageDTO{
		Records:      records,
		Count:        count,
		HasMore:      current*size < count,
		FrameOrigins: append([]string(nil), s.allowedOrigins...),
	})
}

func (s *MyVideoService) Upload(ctx context.Context, input VideoUploadInput) port.ResultVO {
	if s == nil || !s.enabled || s.repository == nil || s.storage == nil {
		return port.ResultFailWithMessage("视频暂未开放上传")
	}
	if input.File == nil {
		return port.ResultFromError(apperrors.Invalid("video.upload", "file is required"))
	}
	if input.File.Size > port.MaxVideoBytes {
		return port.ResultFromError(apperrors.Invalid("video.upload", "video file exceeds 500MB"))
	}
	extension := strings.ToLower(filepath.Ext(input.File.Filename))
	expectedMIME, ok := videoMIMEByExtension[extension]
	if !ok {
		return port.ResultFromError(apperrors.Invalid("video.upload", "video extension is not allowed"))
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(input.File.Filename), filepath.Ext(input.File.Filename))
	}
	if err := validateVideoText(title, input.Description); err != nil {
		return port.ResultFromError(err)
	}
	declaredMIME, err := multipartMIME(input.File)
	if err != nil {
		return port.ResultFromError(apperrors.Invalid("video.upload", "video MIME type is invalid"))
	}
	if declaredMIME != "" && !compatibleVideoMIME(expectedMIME, declaredMIME) {
		return port.ResultFromError(apperrors.Invalid("video.upload", "video MIME type does not match extension"))
	}
	temp, size, detectedMIME, err := copyAndInspectVideo(input.File, extension)
	if err != nil {
		return port.ResultFromError(err)
	}
	defer func() {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
	}()
	if !compatibleVideoMIME(expectedMIME, detectedMIME) {
		return port.ResultFromError(apperrors.Invalid("video.upload", "video content does not match extension"))
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return port.ResultFromError(apperrors.Unavailable("video.upload.seek", err))
	}
	key := "videos/" + uuid.NewString() + extension
	ref, err := s.storage.Put(ctx, key, temp)
	if err != nil {
		return port.ResultFromError(err)
	}
	if err := validateObjectRef(ref, key); err != nil {
		return port.ResultFromError(err)
	}
	video := port.Video{
		ID:          uuid.NewString(),
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		Source:      port.VideoSourceLocal,
		URL:         ref.URL,
		MIMEType:    detectedMIME,
		SizeBytes:   size,
		Published:   true,
		CreatedAt:   s.currentTime(),
		UpdatedAt:   s.currentTime(),
	}
	if err := port.ValidateVideo(video); err != nil {
		return port.ResultFromError(apperrors.Invalid("video.upload", err.Error()))
	}
	if err := s.repository.Create(ctx, video); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(toVideoDTO(video))
}

func (s *MyVideoService) CreateExternal(ctx context.Context, input ExternalVideoInput) port.ResultVO {
	if s == nil || !s.enabled || s.repository == nil {
		return port.ResultFailWithMessage("视频暂未开放管理")
	}
	videoURL, err := validateExternalVideoURL(input.URL, s.allowedOrigins)
	if err != nil {
		return port.ResultFromError(err)
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return port.ResultFromError(apperrors.Invalid("video.external", "title is required"))
	}
	if err := validateVideoText(title, input.Description); err != nil {
		return port.ResultFromError(err)
	}
	video := port.Video{
		ID:          uuid.NewString(),
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		Source:      port.VideoSourceExternal,
		URL:         videoURL,
		EmbedURL:    videoURL,
		Published:   true,
		CreatedAt:   s.currentTime(),
		UpdatedAt:   s.currentTime(),
	}
	if err := port.ValidateVideo(video); err != nil {
		return port.ResultFromError(apperrors.Invalid("video.external", err.Error()))
	}
	if err := s.repository.Create(ctx, video); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(toVideoDTO(video))
}

func (s *MyVideoService) Delete(ctx context.Context, id string) port.ResultVO {
	if s == nil || !s.enabled || s.repository == nil {
		return port.ResultFailWithMessage("视频暂未开放管理")
	}
	id = strings.TrimSpace(id)
	if id == "" || len(id) > port.MaxVideoIDLength || strings.ContainsAny(id, "\r\n\x00") {
		return port.ResultFromError(apperrors.Invalid("video.delete", "video id is invalid"))
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (s *MyVideoService) AllowedFrameOrigins() []string {
	if s == nil {
		return nil
	}
	return append([]string(nil), s.allowedOrigins...)
}

func (s *MyVideoService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func normalizeVideoQuery(query VideoQuery) (int, int, error) {
	current, err := parsePositivePage(query.Current, 1, "video.current")
	if err != nil {
		return 0, 0, err
	}
	size, err := parsePositivePage(query.Size, defaultVideoPageSize, "video.size")
	if err != nil {
		return 0, 0, err
	}
	if size > maxVideoPageSize {
		size = maxVideoPageSize
	}
	return current, size, nil
}

func validateVideoText(title, description string) error {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > port.MaxVideoTitleRunes || strings.ContainsAny(title, "\r\n\x00") {
		return apperrors.Invalid("video.metadata", "video title is invalid")
	}
	if len([]rune(description)) > port.MaxVideoDescriptionRunes || strings.ContainsAny(description, "\x00") {
		return apperrors.Invalid("video.metadata", "video description is invalid")
	}
	return nil
}

func parsePositivePage(value string, fallback int, operation string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, apperrors.Invalid(operation, "page value must be a positive integer")
	}
	return parsed, nil
}

func multipartMIME(file *multipart.FileHeader) (string, error) {
	if file == nil || file.Header == nil {
		return "", nil
	}
	raw := strings.TrimSpace(file.Header.Get("Content-Type"))
	if raw == "" {
		return "", nil
	}
	parsed, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(parsed)), nil
}

func copyAndInspectVideo(fileHeader *multipart.FileHeader, extension string) (*os.File, int64, string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, 0, "", apperrors.Unavailable("video.upload.open", err)
	}
	defer file.Close()
	temp, err := os.CreateTemp("", "benetnasch-video-*")
	if err != nil {
		return nil, 0, "", apperrors.Unavailable("video.upload.temp", err)
	}
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = temp.Close()
			_ = os.Remove(temp.Name())
		}
	}()
	size, err := io.Copy(temp, io.LimitReader(file, port.MaxVideoBytes+1))
	if err != nil {
		return nil, 0, "", apperrors.Unavailable("video.upload.read", err)
	}
	if size <= 0 || size > port.MaxVideoBytes {
		return nil, 0, "", apperrors.Invalid("video.upload", "video file must be between 1 byte and 500MB")
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return nil, 0, "", apperrors.Unavailable("video.upload.inspect", err)
	}
	header := make([]byte, 512)
	read, err := io.ReadFull(temp, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, 0, "", apperrors.Unavailable("video.upload.inspect", err)
	}
	if read == 0 {
		return nil, 0, "", apperrors.Invalid("video.upload", "video header is empty")
	}
	detected := detectVideoMIME(header[:read], extension)
	if detected == "" {
		return nil, 0, "", apperrors.Invalid("video.upload", "video signature is not recognized")
	}
	removeTemp = false
	return temp, size, detected, nil
}

func detectVideoMIME(header []byte, extension string) string {
	if len(header) >= 4 && string(header[:4]) == "OggS" {
		return "video/ogg"
	}
	if len(header) >= 4 && header[0] == 0x1A && header[1] == 0x45 && header[2] == 0xDF && header[3] == 0xA3 {
		return "video/webm"
	}
	if len(header) >= 12 && string(header[4:8]) == "ftyp" {
		brand := string(header[8:12])
		if brand == "qt  " || extension == ".mov" {
			return "video/quicktime"
		}
		return "video/mp4"
	}
	return ""
}

func compatibleVideoMIME(expected, actual string) bool {
	return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(actual)) ||
		(expected == "video/quicktime" && actual == "video/mp4") ||
		(expected == "video/mp4" && actual == "video/quicktime")
}

func validateObjectRef(ref port.ObjectRef, expectedKey string) error {
	if strings.TrimSpace(ref.Key) == "" || ref.Key != expectedKey || strings.ContainsAny(ref.Key, "\r\n\x00") {
		return apperrors.Unavailable("video.upload.storage", nil)
	}
	parsed, err := url.Parse(strings.TrimSpace(ref.URL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return apperrors.Unavailable("video.upload.storage", nil)
	}
	return nil
}

func normalizeVideoOrigins(values []string) ([]string, error) {
	origins := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimRight(strings.TrimSpace(value), "/")
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, apperrors.Invalid("video.origins", "video origins must be HTTPS origins")
		}
		value = "https://" + strings.ToLower(parsed.Host)
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		origins = append(origins, value)
	}
	return origins, nil
}

func validateExternalVideoURL(raw string, origins []string) (string, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || strings.ContainsAny(parsed.String(), "\r\n\x00") {
		return "", apperrors.Invalid("video.external.url", "external video URL must be HTTPS")
	}
	origin := "https://" + strings.ToLower(parsed.Host)
	for _, allowed := range origins {
		if strings.EqualFold(origin, allowed) {
			return parsed.String(), nil
		}
	}
	return "", apperrors.New(apperrors.KindForbidden, "video.external.origin", nil)
}

func toVideoDTO(video port.Video) port.VideoDTO {
	return port.VideoDTO{
		ID:          video.ID,
		Title:       video.Title,
		Description: video.Description,
		Source:      string(video.Source),
		URL:         video.URL,
		EmbedURL:    video.EmbedURL,
		MIMEType:    video.MIMEType,
		SizeBytes:   video.SizeBytes,
		CreatedAt:   video.CreatedAt.UTC(),
		UpdatedAt:   video.UpdatedAt.UTC(),
	}
}

var _ VideoService = (*MyVideoService)(nil)
