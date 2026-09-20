package port

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
)

type CommentRepository interface {
	ListComments(ctx context.Context, filter CommentFilter) ([]*Comment, int, error)
	ResolveCommentPage(ctx context.Context, commentType, topicID, commentID, size int) (int, error)
	ListReplies(ctx context.Context, commentIDs []int) ([]*Reply, error)
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
}
