package api

import "testing"

func TestJobLogStatusRejectsFractionalNumbers(t *testing.T) {
	if _, ok := jobLogStatus(float64(1.5)); ok {
		t.Fatal("fractional status must be rejected")
	}
}
