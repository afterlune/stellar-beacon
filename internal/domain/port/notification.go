package port

import "time"

const (
	NotificationGroupAll      = "all"
	NotificationGroupPublish  = "publish"
	NotificationGroupComment  = "comment"
	NotificationGroupReaction = "reaction"
)

const (
	NotificationTypePublish  = "publish"
	NotificationTypeComment  = "comment"
	NotificationTypeReply    = "reply"
	NotificationTypeLike     = "like"
	NotificationTypeFavorite = "favorite"
)

func ValidNotificationGroup(value string) bool {
	switch value {
	case NotificationGroupAll, NotificationGroupPublish, NotificationGroupComment, NotificationGroupReaction:
		return true
	default:
		return false
	}
}

type NotificationItem struct {
	Key         string       `json:"key"`
	Type        string       `json:"type"`
	Group       string       `json:"group"`
	Actor       PublicAuthor `json:"actor"`
	ContentType string       `json:"contentType"`
	ContentId   int          `json:"contentId"`
	CommentId   int          `json:"commentId,omitempty"`
	Title       string       `json:"title"`
	Excerpt     string       `json:"excerpt"`
	Cover       string       `json:"cover,omitempty"`
	Images      []string     `json:"images,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	Read        bool         `json:"read"`
}

type NotificationCursor struct {
	PublishEventId int64 `json:"publishEventId"`
	InteractionId  int64 `json:"interactionId"`
}

type NotificationPage struct {
	Records          []NotificationItem `json:"records"`
	Count            int                `json:"count"`
	UnreadCount      int                `json:"unreadCount"`
	TotalUnreadCount int                `json:"totalUnreadCount"`
	ReadCursor       NotificationCursor `json:"readCursor"`
}
