package port

import (
	"context"
	"time"
)

type ContentAuditFilter struct {
	Current     int
	Size        int
	ContentType string
	Operation   string
	Result      string
	Keyword     string
	StartDate   time.Time
	EndDate     time.Time
}

type ContentAuditRecord struct {
	ID               int       `json:"id"`
	OperatorID       int       `json:"operatorId"`
	OperatorNickname string    `json:"operatorNickname"`
	ContentType      string    `json:"contentType"`
	Operation        string    `json:"operation"`
	TargetMode       string    `json:"targetMode"`
	FilterSnapshot   string    `json:"filterSnapshot"`
	SnapshotMaxID    int       `json:"snapshotMaxId"`
	RequestedCount   int       `json:"requestedCount"`
	AffectedCount    int       `json:"affectedCount"`
	Result           string    `json:"result"`
	ErrorMessage     string    `json:"errorMessage,omitempty"`
	IPAddress        string    `json:"ipAddress"`
	IPSource         string    `json:"ipSource"`
	CreateTime       time.Time `json:"createTime"`
}

type ContentAuditItem struct {
	ID             int    `json:"id"`
	ContentID      int    `json:"contentId"`
	Title          string `json:"title"`
	PreviousStatus int    `json:"previousStatus"`
	NextStatus     int    `json:"nextStatus"`
	Result         string `json:"result"`
}

type ContentAuditRepository interface {
	List(ctx context.Context, filter ContentAuditFilter) ([]ContentAuditRecord, int64, error)
	ListItems(ctx context.Context, auditID, current, size int) ([]ContentAuditItem, int64, error)
}
