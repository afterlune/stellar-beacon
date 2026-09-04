package port

import "testing"

func TestIsPublicArticleFailsClosed(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		isDelete int
		want     bool
	}{
		{name: "public", status: PublicArticleStatus, isDelete: 0, want: true},
		{name: "private", status: 2, isDelete: 0},
		{name: "draft", status: 3, isDelete: 0},
		{name: "deleted public", status: PublicArticleStatus, isDelete: 1},
		{name: "unknown deletion state", status: PublicArticleStatus, isDelete: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsPublicArticle(test.status, test.isDelete); got != test.want {
				t.Fatalf("IsPublicArticle(%d, %d) = %v, want %v", test.status, test.isDelete, got, test.want)
			}
		})
	}
}
