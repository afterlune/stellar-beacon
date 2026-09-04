package shared

import (
	apperrors "benetnasch/app/domain/errors"
	"log/slog"
	"math/rand"
	"time"

	"github.com/bwmarrin/snowflake"
)

const (
	FILEURL = "http://i.example.invalid/"
)

const (
	ONE               = 1
	ZERO              = 0
	FALSE             = 0
	TRUE              = 1
	BLOGGER_ID        = 1
	DEFAULT_CONFIG_ID = 1
	DEFAULT_ABOUT_ID  = 1
	PRE_TAG           = "<mark>"
	POST_TAG          = "</mark>"
	CURRENT           = "current"
	SIZE              = "size"
	DEFAULT_SIZE      = "10"
	DEFAULT_NICKNAME  = "用户"
	COMPONENT         = "Layout"
	UNKNOWN           = "未知"
	APPLICATION_JSON  = "application/json;charset=utf-8"
	CAPTCHA           = "验证码"
	CHECK_REMIND      = "审核提醒"
	COMMENT_REMIND    = "评论提醒"
	MENTION_REMIND    = "@提醒"
)

const (
	TWENTY_MINUTES = 20 * time.Minute

	EXPIRE_TIME         = 7 * 24 * time.Hour
	REFRESH_EXPIRE_TIME = 30 * 24 * time.Hour

	TOKEN_HEADER         = "Authorization"
	REFRESH_TOKEN_HEADER = "X-Refresh-Token"

	TOKEN_PREFIX = "Bearer "

	ACCESS_LIMIT = 60
)

const (
	TOKEN_BLACKLIST      = "token_blacklist"
	REFRESH_TOKEN_PREFIX = "refresh_token_"
)

const (
	ARTICLE = iota + 1

	MESSAGE

	ABOUTS

	LINK

	TALK
)

var (
	TypeHM = map[int]map[string]string{
		1: {"desc": "文章", "path": "/articles/"},
		2: {"desc": "留言", "path": "/message/"},
		3: {"desc": "关于我", "path": "/about/"},
		4: {"desc": "友链", "path": "/friends/"},
		5: {"desc": "说说", "path": "/talks/"},
	}
	SECRET = ""
)

func init() {
	nod, err := snowflake.NewNode(rand.Int63n(1023)) // 传入节点ID
	if err != nil {
		slog.Error("initialize snowflake node failed", "error_code", apperrors.SafeCode(err))
		return
	}
	SECRET = nod.Generate().String()
}
