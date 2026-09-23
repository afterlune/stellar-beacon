package port

import (
	"time"
)

// ArticleCard is the application-facing read model for article cards. It is
// deliberately independent of the HTTP facade package.
type PublicAuthor struct {
	Id       int    `json:"id"`
	Handle   string `json:"handle"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Intro    string `json:"intro"`
	Website  string `json:"website"`
}
type ArticleCard struct {
	Id                 int          `json:"id"`
	UserId             int          `json:"userId"`
	ArticleCover       string       `json:"articleCover"`
	ArticleTitle       string       `json:"articleTitle"`
	ArticleContent     string       `json:"articleContent"`
	ArticleContentHTML string       `json:"articleContentHtml,omitempty"`
	IsTop              int          `json:"isTop"`
	IsFeatured         int          `json:"isFeatured"`
	CategoryName       string       `json:"categoryName"`
	Status             int          `json:"status"`
	ModerationStatus   string       `json:"moderationStatus"`
	ModerationReason   string       `json:"moderationReason,omitempty"`
	CreateTime         time.Time    `json:"createTime"`
	UpdateTime         time.Time    `json:"updateTime"`
	Author             PublicAuthor `json:"author"`
	Tags               any          `json:"tags"`
	LikeCount          int          `json:"likeCount"`
	FavoriteCount      int          `json:"favoriteCount"`
}

type Article struct {
	Id                 int           `json:"id"`
	UserId             int           `json:"userId"`
	ArticleCover       string        `json:"articleCover"`
	ArticleTitle       string        `json:"articleTitle"`
	ArticleContent     string        `json:"articleContent"`
	ArticleContentHTML string        `json:"articleContentHtml,omitempty"`
	IsTop              int           `json:"isTop"`
	IsFeatured         int           `json:"isFeatured"`
	CategoryName       string        `json:"categoryName"`
	Status             int           `json:"status"`
	ModerationStatus   string        `json:"moderationStatus"`
	ModerationReason   string        `json:"moderationReason,omitempty"`
	CreateTime         time.Time     `json:"createTime"`
	UpdateTime         time.Time     `json:"updateTime"`
	Author             PublicAuthor  `json:"author"`
	Type               int           `json:"type"`
	OriginalUrl        string        `json:"originalUrl"`
	IsDelete           int           `json:"isDelete"`
	ViewCount          int           `json:"viewCount"`
	PreArticleCard     ArticleCard   `json:"preArticleCard"`
	NextArticleCard    ArticleCard   `json:"nextArticleCard"`
	RelatedArticles    []ArticleCard `json:"relatedArticles"`
	Tags               any           `json:"tags"`
	LikeCount          int           `json:"likeCount"`
	FavoriteCount      int           `json:"favoriteCount"`
	SeriesId           int           `json:"seriesId"`
	SeriesOrder        int           `json:"seriesOrder"`
}

type ArticleAdmin struct {
	Id               int        `json:"id"`
	UserId           int        `json:"userId"`
	ArticleCover     string     `json:"articleCover"`
	ArticleTitle     string     `json:"articleTitle"`
	SeriesID         int        `json:"seriesId"`
	SeriesOrder      int        `json:"seriesOrder"`
	ScheduledAt      *time.Time `json:"scheduledAt,omitempty"`
	IsTop            int        `json:"isTop"`
	IsFeatured       int        `json:"isFeatured"`
	IsDelete         int        `json:"isDelete"`
	Status           int        `json:"status"`
	Type             int        `json:"type"`
	ModerationStatus string     `json:"moderationStatus"`
	ModerationReason string     `json:"moderationReason,omitempty"`
	ModeratedBy      int        `json:"moderatedBy,omitempty"`
	ModeratedAt      *time.Time `json:"moderatedAt,omitempty"`
	AuthorHandle     string     `json:"authorHandle"`
	AuthorNickname   string     `json:"authorNickname"`
	AuthorAvatar     string     `json:"authorAvatar"`
	CreateTime       time.Time  `json:"createTime"`
	CategoryName     string     `json:"categoryName"`
	ViewsCount       int        `json:"viewsCount"`
	TagDTOs          []Tag      `json:"tagDTOs"`
	LikeCount        int        `json:"likeCount"`
	FavoriteCount    int        `json:"favoriteCount"`
}

type ArticleAdminView struct {
	Id                 int    `json:"id"`
	UserId             int    `json:"userId"`
	ArticleCover       string `json:"articleCover"`
	ArticleTitle       string `json:"articleTitle"`
	ArticleContent     string `json:"articleContent"`
	ArticleContentHTML string `json:"articleContentHtml,omitempty"`
	IsTop              int    `json:"isTop"`
	IsFeatured         int    `json:"isFeatured"`
	CategoryId         int    `json:"categoryId"`
	CategoryName       string `json:"categoryName"`
	TagNames           any    `json:"tagNames"`
	Status             int    `json:"status"`
	Type               int    `json:"type"`
	Password           string `json:"password"`
	OriginalUrl        string `json:"originalUrl"`
	SeriesId           int    `json:"seriesId"`
	SeriesOrder        int    `json:"seriesOrder"`
	ScheduledAt        string `json:"scheduledAt"`
	ModerationStatus   string `json:"moderationStatus"`
	ModerationReason   string `json:"moderationReason,omitempty"`
}

type ArticleSearch struct {
	Id             int    `json:"id"`
	ArticleTitle   string `json:"articleTitle"`
	ArticleContent string `json:"articleContent"`
	IsDelete       int    `json:"isDelete"`
	Status         int    `json:"status"`
}

type ArticleStatistics struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Tag struct {
	Id      int    `json:"id"`
	UserId  int    `json:"userId"`
	TagName string `json:"tagName"`
	Count   int    `json:"count"`
}

type TagAdmin struct {
	Id           int       `json:"id"`
	UserId       int       `json:"userId"`
	TagName      string    `json:"tagName"`
	ArticleCount int       `json:"articleCount"`
	CreateTime   time.Time `json:"createTime"`
}

type TagFilter struct {
	Keywords string
}

type Category struct {
	Id           int    `json:"id"`
	UserId       int    `json:"userId"`
	CategoryName string `json:"categoryName"`
	ArticleCount int    `json:"articleCount"`
}

type CategoryAdmin struct {
	Id           int       `json:"id"`
	UserId       int       `json:"userId"`
	CategoryName string    `json:"categoryName"`
	ArticleCount string    `json:"articleCount"`
	CreateTime   time.Time `json:"createTime"`
}

type CategoryOption struct {
	Id           int    `json:"id"`
	CategoryName string `json:"categoryName"`
}

type CategoryFilter struct {
	Keywords string
}

type ArticleFilter struct {
	Current          int
	Size             int
	Keywords         string
	IsDelete         int
	Status           int
	ModerationStatus string
	UserId           int
	Category         int
	Type             int
	Tag              int
}

type Comment struct {
	Id             int       `json:"id"`
	UserId         int       `json:"userId"`
	Nickname       string    `json:"nickname"`
	Avatar         string    `json:"avatar"`
	Website        string    `json:"website"`
	CommentContent string    `json:"commentContent"`
	CreateTime     time.Time `json:"createTime"`
	LikeCount      int       `json:"likeCount"`
	Liked          bool      `json:"liked"`
	IsTop          int       `json:"isTop"`
	ReplyDTOs      []Reply   `json:"replyDTOs"`
	IsDelete       int       `json:"isDelete"`
}

type Reply struct {
	Id             int       `json:"id"`
	ParentId       int       `json:"parentId"`
	UserId         int       `json:"userId"`
	Nickname       string    `json:"nickname"`
	Avatar         string    `json:"avatar"`
	Website        string    `json:"website"`
	ReplyUserId    int       `json:"replyUserId"`
	ReplyNickname  string    `json:"replyNickname"`
	ReplyWebsite   string    `json:"replyWebsite"`
	CommentContent string    `json:"commentContent"`
	CreateTime     time.Time `json:"createTime"`
	LikeCount      int       `json:"likeCount"`
	Liked          bool      `json:"liked"`
	IsDelete       int       `json:"isDelete"`
}

type CommentAdmin struct {
	Id             int       `json:"id"`
	Avatar         string    `json:"avatar"`
	Nickname       string    `json:"nickname"`
	ReplyNickname  string    `json:"replyNickname"`
	ArticleTitle   string    `json:"articleTitle"`
	CommentContent string    `json:"commentContent"`
	Type           int       `json:"type"`
	IsReview       int       `json:"isReview"`
	IsTop          int       `json:"isTop"`
	IsDelete       int       `json:"isDelete"`
	CollectionId   int       `json:"collectionId"`
	ReportCount    int       `json:"reportCount"`
	CreateTime     time.Time `json:"createTime"`
}

type OwnedComment struct {
	Id             int       `json:"id"`
	UserId         int       `json:"userId"`
	Nickname       string    `json:"nickname"`
	Avatar         string    `json:"avatar"`
	CommentContent string    `json:"commentContent"`
	ParentId       int       `json:"parentId"`
	IsTop          int       `json:"isTop"`
	IsDelete       int       `json:"isDelete"`
	IsReview       int       `json:"isReview"`
	ReplyCount     int       `json:"replyCount"`
	ReportCount    int       `json:"reportCount"`
	CreateTime     time.Time `json:"createTime"`
}

type ModerationFailure struct {
	CommentId int    `json:"commentId"`
	Message   string `json:"message"`
}

type ModerationBatchResult struct {
	Succeeded []int               `json:"succeeded"`
	Failed    []ModerationFailure `json:"failed"`
}
type CommentCount struct {
	Id           int `json:"id"`
	CommentCount int `json:"commentCount"`
}

type CommentFilter struct {
	Current      int
	Size         int
	Keywords     string
	Type         int
	IsReview     int
	TopicID      *int
	CollectionID int
	ViewerID     int
}

type UserFilter struct {
	Current   int
	Size      int
	Keywords  string
	LoginType int
}

type UserAdmin struct {
	Id            int        `json:"id"`
	UserInfoId    int        `json:"userInfoId"`
	Avatar        string     `json:"avatar"`
	Nickname      string     `json:"nickname"`
	LoginType     int        `json:"loginType"`
	IpAddress     string     `json:"ipAddress"`
	IpSource      string     `json:"ipSource"`
	CreateTime    time.Time  `json:"createTime"`
	LastLoginTime time.Time  `json:"lastLoginTime"`
	IsDisable     int        `json:"isDisable"`
	Status        int        `json:"status"`
	Roles         []UserRole `json:"roles"`
}

type UserRole struct {
	Id       int    `json:"id"`
	RoleName string `json:"roleName"`
}

type Talk struct {
	Id               int       `json:"id"`
	UserId           int       `json:"userId"`
	Handle           string    `json:"handle"`
	Nickname         string    `json:"nickName"`
	Avatar           string    `json:"avatar"`
	Content          string    `json:"content"`
	Images           string    `json:"images"`
	Imgs             []string  `json:"imgs"`
	IsTop            int       `json:"isTop"`
	Status           int       `json:"status"`
	ModerationStatus string    `json:"moderationStatus"`
	ModerationReason string    `json:"moderationReason,omitempty"`
	CommentCount     int       `json:"commentCount"`
	CreateTime       time.Time `json:"createTime"`
}

type TalkAdmin struct {
	Id     int `json:"id"`
	UserId int `json:"userId"`

	Nickname         string     `json:"nickname"`
	Avatar           string     `json:"avatar"`
	Content          string     `json:"content"`
	Images           string     `json:"images"`
	Imgs             []string   `json:"imgs"`
	IsTop            int        `json:"isTop"`
	Status           int        `json:"status"`
	ModerationStatus string     `json:"moderationStatus"`
	ModerationReason string     `json:"moderationReason,omitempty"`
	ModeratedBy      int        `json:"moderatedBy,omitempty"`
	ModeratedAt      *time.Time `json:"moderatedAt,omitempty"`
	Handle           string     `json:"authorHandle"`
	CreateTime       time.Time  `json:"createTime"`
}

type TalkFilter struct {
	Status           int
	ModerationStatus string
	UserId           int
	Keywords         string
}
