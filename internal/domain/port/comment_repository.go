package port

import (
	"context"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"time"
)

type CommentRepository interface {
	ListComments(ctx context.Context, filter CommentFilter) ([]*Comment, int, error)
	ResolveCommentPage(ctx context.Context, commentType, topicID, commentID, size int) (int, error)
	ListReplies(ctx context.Context, commentIDs []int, viewerID int) ([]*Reply, error)
	ListTopSixComments(ctx context.Context) ([]*Comment, error)
	CountComments(ctx context.Context, filter CommentFilter) (int64, error)
	ListCommentsAdmin(ctx context.Context, filter CommentFilter) ([]*CommentAdmin, error)
	ListCommentCountsByTypeAndTopicIDs(ctx context.Context, commentType int, topicIDs []int) ([]*CommentCount, error)
	ListCommentCountByTypeAndTopicID(ctx context.Context, commentType, topicID int) (CommentCount, error)
	ValidateTarget(ctx context.Context, commentType, topicID int) error
	ValidateReply(ctx context.Context, commentType, parentID, replyUserID int) error
	GetByID(ctx context.Context, commentID int) (entity.TComment, error)
	Create(ctx context.Context, comment entity.TComment) (int, error)
	MarkNotificationDispatched(ctx context.Context, commentID int) error
	Review(ctx context.Context, ids []int, review int) error
	Delete(ctx context.Context, ids []int) error
	SetPinned(ctx context.Context, userID, collectionID, commentID int, pinned bool) error
	SoftDeleteOwned(ctx context.Context, userID, collectionID, commentID int) error
	BatchModerateOwned(ctx context.Context, userID, collectionID int, action string, commentIDs []int) (ModerationBatchResult, error)
	RestoreOwned(ctx context.Context, userID, collectionID int, commentIDs []int) (ModerationBatchResult, error)
	RestoreAsAdmin(ctx context.Context, adminID, collectionID int, commentIDs []int) (ModerationBatchResult, error)
	ListOwnedCollectionComments(ctx context.Context, userID, collectionID, current, size int, keywords string, includeDeleted bool) ([]*OwnedComment, int, error)
	CountPendingReportsByCommentIDs(ctx context.Context, commentIDs []int) (map[int]int, error)
	CreateCommentReport(ctx context.Context, commentID, collectionID, reporterID int, reason, detail string) error
	ListOwnerCommentReports(ctx context.Context, ownerID, collectionID, current, size int) ([]*CommentReportGroup, int, error)
	ListAdminCommentReports(ctx context.Context, current, size int) ([]*CommentReportGroup, int, error)
	ResolveCommentReports(ctx context.Context, actorID int, actorRole string, collectionID, commentID int, decision, reason string) ([]int, error)
	CreateCommentAppeal(ctx context.Context, appellantID, commentID int, reason string) error
	ListOwnerCommentAppeals(ctx context.Context, ownerID, collectionID, current, size int) ([]*CommentAppealItem, int, error)
	ListAdminCommentAppeals(ctx context.Context, current, size int) ([]*CommentAppealItem, int, error)
	ListMyCommentAppeals(ctx context.Context, appellantID, current, size int) ([]*CommentAppealItem, int, error)
	ResolveCommentAppeal(ctx context.Context, actorID int, actorRole string, appealID int, decision, reason string) (CommentAppealOutcome, error)
	EscalateCommentAppeal(ctx context.Context, appellantID, appealID int) error
	CreateModerationNotification(ctx context.Context, recipientID, actorID int, contentType string, contentID, commentID int, dedupeKey string) error
	WriteModerationAudit(ctx context.Context, entry ModerationAuditEntry) error
}

// ModerationAuditEntry describes one append-only governance event. It is used by
// the report and appeal services, which own their own tables but still record
// every state change in the shared moderation ledger.
type ModerationAuditEntry struct {
	CommentID    int
	CollectionID int
	ActorID      int
	ActorRole    string
	Action       string
	Reason       string
}

// CommentReportGroup is one reading-list comment with its pending reader reports
// collapsed into a single moderation queue entry.
type CommentReportGroup struct {
	CommentId        int       `json:"commentId"`
	CollectionId     int       `json:"collectionId"`
	CollectionTitle  string    `json:"collectionTitle"`
	CommentContent   string    `json:"commentContent"`
	CommentNickname  string    `json:"commentNickname"`
	CommentIsDelete  int       `json:"commentIsDelete"`
	ReportCount      int       `json:"reportCount"`
	Reasons          string    `json:"reasons"`
	LatestDetail     string    `json:"latestDetail"`
	LatestCreateTime time.Time `json:"latestCreateTime"`
}

// CommentAppealOutcome identifies the accounts interested in an appeal decision.
type CommentAppealOutcome struct {
	CommentID         int
	CollectionID      int
	AppellantID       int
	CollectionOwnerID int
}

// CommentAppealItem is one appeal from a reader whose comment was hidden.
type CommentAppealItem struct {
	Id              int       `json:"id"`
	CommentId       int       `json:"commentId"`
	CollectionId    int       `json:"collectionId"`
	CollectionTitle string    `json:"collectionTitle"`
	CommentContent  string    `json:"commentContent"`
	AppellantId     int       `json:"appellantId"`
	AppellantName   string    `json:"appellantName"`
	Reason          string    `json:"reason"`
	Stage           string    `json:"stage"`
	Status          string    `json:"status"`
	DecisionReason  string    `json:"decisionReason"`
	CreateTime      time.Time `json:"createTime"`
}
