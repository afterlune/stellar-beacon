package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"benetnasch/app/domain/port"
	"github.com/google/uuid"
)

const (
	defaultDreamImagePollInterval = 10 * time.Second
	defaultDreamImageLease        = 5 * time.Minute
	defaultDreamPlaceholder       = "/dream-placeholder.svg"
)

type DreamImageWorkerDeps struct {
	Dreams         port.DreamRepository
	Images         port.ImageGateway
	Safety         port.AgentSafetySwitch
	PlaceholderURL string
	WorkerID       string
	PollInterval   time.Duration
	LeaseDuration  time.Duration
}

// DreamImageWorker is optional and deliberately provider-agnostic. With no
// ImageGateway configured, or when a provider fails validation, it records a
// local placeholder instead of blocking the approved dream feed.
type DreamImageWorker struct {
	dreams         port.DreamRepository
	images         port.ImageGateway
	safety         port.AgentSafetySwitch
	placeholderURL string
	workerID       string
	pollInterval   time.Duration
	leaseDuration  time.Duration
	now            func() time.Time
}

var _ Worker = (*DreamImageWorker)(nil).Run

func NewDreamImageWorker(deps DreamImageWorkerDeps) (*DreamImageWorker, error) {
	if deps.Dreams == nil {
		return nil, errors.New("dream image worker requires dream repository")
	}
	deps.PlaceholderURL = strings.TrimSpace(deps.PlaceholderURL)
	if deps.PlaceholderURL == "" {
		deps.PlaceholderURL = defaultDreamPlaceholder
	}
	if len(deps.PlaceholderURL) > 2048 || strings.ContainsAny(deps.PlaceholderURL, "\r\n\x00") {
		return nil, errors.New("dream image placeholder url is invalid")
	}
	if strings.TrimSpace(deps.WorkerID) == "" {
		deps.WorkerID = "agent-dream-image-" + uuid.NewString()
	}
	if deps.PollInterval <= 0 {
		deps.PollInterval = defaultDreamImagePollInterval
	}
	if deps.LeaseDuration <= 0 {
		deps.LeaseDuration = defaultDreamImageLease
	}
	return &DreamImageWorker{
		dreams:         deps.Dreams,
		images:         deps.Images,
		safety:         deps.Safety,
		placeholderURL: deps.PlaceholderURL,
		workerID:       strings.TrimSpace(deps.WorkerID),
		pollInterval:   deps.PollInterval,
		leaseDuration:  deps.LeaseDuration,
		now:            func() time.Time { return time.Now().UTC() },
	}, nil
}

func (w *DreamImageWorker) Run(ctx context.Context) error {
	if w == nil {
		return errors.New("dream image worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		if w.safety != nil {
			stopped, err := w.safety.IsStopped(ctx)
			if err != nil {
				slog.Error("dream image safety switch read failed", "worker", w.workerID, "error", safeWorkerError(err))
				if !w.wait(ctx) {
					return ctx.Err()
				}
				continue
			}
			if stopped {
				if !w.wait(ctx) {
					return ctx.Err()
				}
				continue
			}
		}
		processed, err := w.processOne(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			slog.Error("dream image worker iteration failed", "worker", w.workerID, "error", safeWorkerError(err))
		}
		if processed {
			continue
		}
		if !w.wait(ctx) {
			return ctx.Err()
		}
	}
}

// RunOnce processes at most one pending dream image. It never creates a new
// image task and remains bounded by the runner context.
func (w *DreamImageWorker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil {
		return false, errors.New("dream image worker is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := checkOneShotSafety(ctx, w.safety); err != nil {
		return false, err
	}
	return w.processOne(ctx)
}

func (w *DreamImageWorker) wait(ctx context.Context) bool {
	timer := time.NewTimer(w.pollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *DreamImageWorker) processOne(ctx context.Context) (bool, error) {
	entries, err := w.dreams.ListPendingImages(ctx, 1, w.currentTime())
	if err != nil {
		return false, err
	}
	if len(entries) == 0 {
		return false, nil
	}
	entry, claimed, err := w.dreams.ClaimImage(ctx, entries[0].ID, w.workerID, w.currentTime(), w.leaseDuration)
	if err != nil {
		return false, err
	}
	if !claimed {
		return true, nil
	}
	status := port.DreamImagePlaceholder
	imageURL := w.placeholderURL
	imageError := "image provider is not configured"
	if w.images != nil {
		response, generateErr := w.images.Generate(ctx, port.ImageRequest{
			Prompt: entry.ImagePrompt,
			Metadata: map[string]string{
				"dream_id":  entry.ID,
				"review_id": entry.ReviewID,
			},
		})
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		if generateErr == nil && validDreamImageURL(response.URL) {
			status = port.DreamImageReady
			imageURL = strings.TrimSpace(response.URL)
			imageError = ""
		} else if generateErr != nil {
			imageError = "image provider failed: " + safeWorkerError(generateErr)
		} else {
			imageError = "image provider returned an invalid url"
		}
	}
	if err := w.dreams.CompleteImage(ctx, entry.ID, w.workerID, status, imageURL, imageError, w.currentTime()); err != nil {
		return true, fmt.Errorf("complete dream image: %w", err)
	}
	return true, nil
}

func validDreamImageURL(raw string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func (w *DreamImageWorker) currentTime() time.Time {
	if w == nil || w.now == nil {
		return time.Now().UTC()
	}
	return w.now().UTC()
}
