package support

import "testing"

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
