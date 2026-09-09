package shared

import "testing"

func TestUnmarshRejectsUnexpectedCacheValue(t *testing.T) {
	var value struct {
		ID int `json:"id"`
	}
	if err := Unmarsh(123, &value); err == nil {
		t.Fatal("expected non-string cache value to be rejected")
	}
}

func TestUnmarshRejectsCorruptedJSON(t *testing.T) {
	var value struct {
		ID int `json:"id"`
	}
	if err := Unmarsh("{", &value); err == nil {
		t.Fatal("expected corrupted JSON to be rejected")
	}
}
