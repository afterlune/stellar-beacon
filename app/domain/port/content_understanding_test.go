package port

import (
	"encoding/json"
	"testing"
	"time"
)

func TestContentUnderstandingJobPayloadValidation(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	payload, err := NewContentUnderstandingJobPayload(42, now)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ContentUnderstandingJobPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ArticleID != 42 || decoded.SchemaVersion != 1 || !decoded.OccurredAt.Equal(now) {
		t.Fatalf("payload = %+v", decoded)
	}
	for _, invalid := range []ContentUnderstandingJobPayload{
		{ArticleID: 42, OccurredAt: now},
		{SchemaVersion: 1, OccurredAt: now},
		{SchemaVersion: 1, ArticleID: 42},
	} {
		if err := invalid.Validate(); err == nil {
			t.Fatalf("Validate(%+v) returned nil", invalid)
		}
	}
}
