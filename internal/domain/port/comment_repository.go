package port

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
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
