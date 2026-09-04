package port

import "time"

// The T-prefixed records are domain persistence contracts. They intentionally
// contain no ORM tags; xorm-specific row types live in infra/persistence/row.
// The names remain stable for the legacy application contract while the
// repository boundary prevents the ORM from leaking into application code.

type TAbout struct {
	Id         int
	Content    string
	CreateTime time.Time
	UpdateTime time.Time
}

type TArticle struct {
	Id             int       `json:"id"`
	UserId         int       `json:"userId"`
	CategoryId     int       `json:"categoryId"`
	ArticleCover   string    `json:"articleCover"`
	ArticleTitle   string    `json:"articleTitle"`
	ArticleContent string    `json:"articleContent"`
	IsTop          int       `json:"isTop"`
	IsFeatured     int       `json:"isFeatured"`
	IsDelete       int       `json:"isDelete"`
	Status         int       `json:"status"`
	Type           int       `json:"type"`
	Password       string    `json:"password"`
	OriginalUrl    string    `json:"originalUrl"`
	CreateTime     time.Time `json:"createTime"`
	UpdateTime     time.Time `json:"updateTime"`
}

type TArticleTag struct {
	Id        int
	ArticleId int
	TagId     int
}

type TCategory struct {
	Id           int       `json:"id"`
	CategoryName string    `json:"categoryName"`
	CreateTime   time.Time `json:"createTime"`
	UpdateTime   time.Time `json:"updateTime"`
}

type TComment struct {
	Id             int       `json:"id"`
	UserId         int       `json:"userId"`
	TopicId        int       `json:"topicId"`
	CommentContent string    `json:"commentContent"`
	ReplyUserId    int       `json:"replyUserId"`
	ParentId       int       `json:"parentId"`
	Type           int       `json:"type"`
	IsDelete       int       `json:"isDelete"`
	IsReview       int       `json:"isReview"`
	CreateTime     time.Time `json:"createTime"`
	UpdateTime     time.Time `json:"updateTime"`
}

type TExceptionLog struct {
	Id            int
	OptUri        string
	OptMethod     string
	RequestMethod string
	RequestParam  string
	OptDesc       string
	ExceptionInfo string
	IpAddress     string
	IpSource      string
	CreateTime    time.Time
}

type TFriendLink struct {
	Id          int
	LinkName    string
	LinkAvatar  string
	LinkAddress string
	LinkIntro   string
	CreateTime  time.Time
	UpdateTime  time.Time
}

type TJob struct {
	Id             int
	JobName        string
	JobGroup       string
	InvokeTarget   string
	CronExpression string
	MisfirePolicy  int
	Concurrent     int
	Status         int
	CreateTime     time.Time
	UpdateTime     time.Time
	Remark         string
}

type TJobLog struct {
	Id            int
	JobId         int
	JobName       string
	JobGroup      string
	InvokeTarget  string
	JobMessage    string
	Status        int
	ExceptionInfo string
	CreateTime    time.Time
	StartTime     time.Time
	EndTime       time.Time
}

type TMenu struct {
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Component  string    `json:"component"`
	Icon       string    `json:"icon"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
	OrderNum   int       `json:"orderNum"`
	ParentId   int       `json:"parentId"`
	IsHidden   int       `json:"isHidden"`
}

type TOperationLog struct {
	Id            int
	OptModule     string
	OptType       string
	OptUri        string
	OptMethod     string
	OptDesc       string
	RequestParam  string
	RequestMethod string
	ResponseData  string
	UserId        int
	Nickname      string
	IpAddress     string
	IpSource      string
	CreateTime    time.Time
	UpdateTime    time.Time
}

type TPhoto struct {
	Id         int
	AlbumId    int
	PhotoName  string
	PhotoDesc  string
	PhotoSrc   string
	IsDelete   int
	CreateTime time.Time
	UpdateTime time.Time
}

type TPhotoAlbum struct {
	Id         int
	AlbumName  string
	AlbumDesc  string
	AlbumCover string
	IsDelete   int
	Status     int
	CreateTime time.Time
	UpdateTime time.Time
}

type TResource struct {
	Id            int
	ResourceName  string
	Url           string
	RequestMethod string
	ParentId      int
	IsAnonymous   int
	CreateTime    time.Time
	UpdateTime    time.Time
}

type TRole struct {
	Id         int
	RoleName   string
	IsDisable  int
	CreateTime time.Time
	UpdateTime time.Time
}

type TRoleMenu struct {
	Id     int
	RoleId int
	MenuId int
}

type TRoleResource struct {
	Id         int
	RoleId     int
	ResourceId int
}

type TTag struct {
	Id         int       `json:"id"`
	TagName    string    `json:"tagName"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}

type TTalk struct {
	Id         int       `json:"id"`
	UserId     int       `json:"userId"`
	Content    string    `json:"content"`
	Images     string    `json:"images"`
	IsTop      int       `json:"isTop"`
	Status     int       `json:"status"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}

type TUniqueView struct {
	Id         int
	ViewsCount int
	CreateTime time.Time
	UpdateTime time.Time
}

type TUserAuth struct {
	Id            int
	UserInfoId    int
	Username      string
	Password      string
	LoginType     int
	IpAddress     string
	IpSource      string
	CreateTime    time.Time
	UpdateTime    time.Time
	LastLoginTime time.Time
}

type TUserInfo struct {
	Id          int       `json:"id"`
	Email       string    `json:"email"`
	Nickname    string    `json:"nickname"`
	Avatar      string    `json:"avatar"`
	Intro       string    `json:"intro"`
	Website     string    `json:"website"`
	IsSubscribe int       `json:"isSubscribe"`
	IsDisable   int       `json:"isDisable"`
	CreateTime  time.Time `json:"createTime"`
	UpdateTime  time.Time `json:"updateTime"`
}

type TUserRole struct {
	Id     int
	UserId int
	RoleId int
}

type TWebsiteConfig struct {
	Id         int
	Config     string
	CreateTime time.Time
	UpdateTime time.Time
}
