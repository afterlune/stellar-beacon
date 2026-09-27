package port

import "context"

// CommentReactionRepository stores explicit likes for comments and replies.
type CommentReactionRepository interface {
	Set(ctx context.Context, commentID, userInfoID int, active bool) (bool, int, error)
}
