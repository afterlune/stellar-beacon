package service

import "time"

const (
	One                = 1
	Zero               = 0
	False              = 0
	True               = 1
	BloggerID          = 1
	DefaultConfigID    = 1
	DefaultAboutID     = 1
	PreTag             = "<mark>"
	PostTag            = "</mark>"
	Current            = "current"
	Size               = "size"
	DefaultSize        = "10"
	DefaultNickname    = "用户"
	Component          = "Layout"
	Unknown            = "未知"
	ApplicationJSON    = "application/json;charset=utf-8"
	Captcha            = "验证码"
	CheckRemind        = "审核提醒"
	CommentRemind      = "评论提醒"
	MentionRemind      = "@提醒"
	TokenHeader        = "Authorization"
	RefreshTokenHeader = "X-Refresh-Token"
	TokenPrefix        = "Bearer "
	AccessLimit        = 60
	TokenBlacklist     = "token_blacklist"
	RefreshTokenPrefix = "refresh_token_"

	CodeExpireTime     = 1000000000 * 60 * 15
	BlogViewsCount     = "blog_views_count"
	DailyViewsPrefix   = "blog_views_daily:"
	ArticleViewsCount  = "article_views_count"
	WebsiteConfig      = "website_config"
	UserArea           = "user_area"
	VisitorArea        = "visitor_area"
	About              = "about"
	UniqueVisitor      = "unique_visitor"
	DailyVisitorPrefix = "unique_visitor:"
	LoginUser          = "login_user"
	ArticleAccess      = "article_access:"
	UserCodeKey        = "code:"
)

const (
	Article = iota + 1
	Message
	Abouts
	Link
	Talk
	Collection
)

const ProfileWall = 7

const (
	TwentyMinutes     = 20 * time.Minute
	ExpireTime        = 7 * 24 * time.Hour
	RefreshExpireTime = 30 * 24 * time.Hour
)

var TypeHM = map[int]map[string]string{
	Article:     {"desc": "文章", "path": "/articles/"},
	Talk:        {"desc": "说说", "path": "/talks/"},
	Collection:  {"desc": "书单", "path": "/collections/"},
	ProfileWall: {"desc": "作者主页", "path": "/u/"},
}
