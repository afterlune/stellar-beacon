package shared

const (
	ARTICLE = iota + 1

	MESSAGE

	ABOUTS

	LINK

	TALK
)

var TypeHM = map[int]map[string]string{
	1: {"desc": "文章", "path": "/articles/"},
	2: {"desc": "留言", "path": "/message/"},
	3: {"desc": "关于我", "path": "/about/"},
	4: {"desc": "友链", "path": "/friends/"},
	5: {"desc": "说说", "path": "/talks/"},
}
