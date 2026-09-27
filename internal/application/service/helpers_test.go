package service

import (
	"mime/multipart"
	"net/textproto"
	"testing"
)

func TestObjectKeyPreservesExtensionAndPrefix(t *testing.T) {
	key, err := ObjectKey("cover.png", "photos/")
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != len("photos/")+32+len(".png") || key[:len("photos/")] != "photos/" || key[len(key)-4:] != ".png" {
		t.Fatalf("unexpected object key %q", key)
	}
}

func TestObjectKeyRejectsMissingExtension(t *testing.T) {
	if _, err := ObjectKey("cover", "photos/"); err == nil {
		t.Fatal("expected missing extension to be rejected")
	}
}

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

func TestIsImageUploadChecksTypeAndSize(t *testing.T) {
	if !isImageUpload(&multipart.FileHeader{
		Filename: "cover.png",
		Header:   textproto.MIMEHeader{"Content-Type": []string{"application/octet-stream"}},
	}) {
		t.Fatal("expected image extension to be accepted")
	}
	if isImageUpload(&multipart.FileHeader{Filename: "notes.txt"}) {
		t.Fatal("expected non-image extension to be rejected")
	}
	if maxImageUploadBytes != 10<<20 {
		t.Fatalf("unexpected image upload limit: %d", maxImageUploadBytes)
	}
}
