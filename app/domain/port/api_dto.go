package port

import (
	"container/list"
	"time"
)

type AboutDTO struct {
	Content string `json:"content"`
}

type ArchiveDTO struct {
	Time     string           `json:"time"`
	Articles []ArticleCardDTO `json:"articles"`
}

type ArticleAdminDTO = ArticleAdmin
type ArticleCardDTO = ArticleCard
type ArticleAdminViewDTO = ArticleAdminView
type ArticleDTO = Article

type ArticleRankDTO struct {
	ArticleTitle string `json:"articleTitle"`
	ViewsCount   int    `json:"viewsCount"`
}

type ArticleSearchDTO = ArticleSearchResult
type ArticleStatisticsDTO = ArticleStatistics

// AIProviderProbeDTO contains only safe route metadata and bounded timing
// information. Provider credentials and raw error details never cross this
// facade boundary.
type AIProviderProbeDTO struct {
	UseCase             string    `json:"useCase"`
	Provider            string    `json:"provider"`
	Protocol            string    `json:"protocol"`
	Model               string    `json:"model"`
	DataPolicy          string    `json:"dataPolicy,omitempty"`
	Reachable           bool      `json:"reachable"`
	Generate            bool      `json:"generate"`
	Streaming           bool      `json:"streaming"`
	Embedding           bool      `json:"embedding"`
	LatencyMS           int64     `json:"latencyMs"`
	FirstTokenLatencyMS int64     `json:"firstTokenLatencyMs"`
	CheckedAt           time.Time `json:"checkedAt"`
	ErrorCode           string    `json:"errorCode,omitempty"`
}

type BenetnaschAdminInfoDTO struct {
	ViewsCount            int       `json:"viewsCount"`
	MessageCount          int64     `json:"messageCount"`
	UserCount             int64     `json:"userCount"`
	ArticleCount          int64     `json:"articleCount"`
	CategoryDTOs          list.List `json:"categoryDTOs"`
	TagDTOs               list.List `json:"tagDTOs"`
	ArticleStatisticsDTOs list.List `json:"articleStatisticsDTOs"`
	UniqueViewDTOs        list.List `json:"uniqueViewDTOs"`
	ArticleRankDTOs       list.List `json:"articleRankDTOs"`
}

type BenetnaschBackInfoDTO struct {
	ViewsCount            int                    `json:"viewsCount"`
	MessageCount          int                    `json:"messageCount"`
	UserCount             int                    `json:"userCount"`
	ArticleCount          int                    `json:"articleCount"`
	CategoryDTOs          []CategoryDTO          `json:"categoryDTOs"`
	TagDTOs               []TagDTO               `json:"tagDTOs"`
	ArticleStatisticsDTOs []ArticleStatisticsDTO `json:"articleStatisticsDTOs"`
	UniqueViewDTOs        []UniqueViewDTO        `json:"uniqueViewDTOs"`
	ArticleRankDTOs       []ArticleRankDTO       `json:"articleRankDTOs"`
}

type BenetnaschHomeInfoDTO struct {
	ArticleCount    int64            `json:"articleCount"`
	TalkCount       int64            `json:"talkCount"`
	CategoryCount   int64            `json:"categoryCount"`
	TagCount        int64            `json:"tagCount"`
	WebsiteConfigDT WebsiteConfigDTO `json:"websiteConfigDTO"`
	ViewCount       int              `json:"viewCount"`
}

type CategoryAdminDTO = CategoryAdmin
type CategoryDTO = Category
type CategoryOptionDTO = CategoryOption

type CommentAdminDTO = CommentAdmin
type CommentCountDTO = CommentCount
type CommentDTO = Comment

type EmailDTO struct {
	Email      string                 `json:"email"`
	Subject    string                 `json:"subject"`
	CommentMap map[string]interface{} `json:"commentMap"`
	Template   string                 `json:"template"`
}

type ExceptionLogDTO struct {
	Id            int       `json:"id"`
	OptUri        string    `json:"optUri"`
	OptMethod     string    `json:"optMethod"`
	RequestMethod string    `json:"requestMethod"`
	RequestParam  string    `json:"requestParam"`
	OptDesc       string    `json:"optDesc"`
	ExceptionInfo string    `json:"exceptionInfo"`
	IpAddress     string    `json:"ipAddress"`
	IpSource      string    `json:"ipSource"`
	CreateTime    time.Time `json:"createTime"`
}

type FriendLinkAdminDTO struct {
	Id          int       `json:"id"`
	LinkName    string    `json:"linkName"`
	LinkAvatar  string    `json:"linkAvatar"`
	LinkAddress string    `json:"linkAddress"`
	LinkIntro   string    `json:"linkIntro"`
	CreateTime  time.Time `json:"createTime"`
}

type FriendLinkDTO struct {
	Id          int    `json:"id"`
	LinkName    string `json:"linkName"`
	LinkAvatar  string `json:"linkAvatar"`
	LinkAddress string `json:"linkAddress"`
	LinkIntro   string `json:"linkIntro"`
}

type JobDTO struct {
	Id             int       `json:"id"`
	JobName        string    `json:"jobName"`
	JobGroup       string    `json:"jobGroup"`
	InvokeTarget   string    `json:"invokeTarget"`
	CronExpression string    `json:"cronExpression"`
	MisfirePolicy  string    `json:"misfirePolicy"`
	Concurrent     int       `json:"concurrent"`
	Status         int       `json:"status"`
	CreateTime     time.Time `json:"createTime"`
	Remark         string    `json:"remark"`
	NextValidTime  time.Time `json:"nextValidTime"`
	CanRunOnce     bool      `json:"canRunOnce"`
	RunOnceReason  string    `json:"runOnceReason,omitempty"`
}

type JobRunOutcomeDTO struct {
	JobID     int    `json:"jobId"`
	Target    string `json:"target"`
	Processed bool   `json:"processed"`
}

type JobLogDTO struct {
	Id            int       `json:"id"`
	JobId         int       `json:"jobId"`
	JobName       string    `json:"jobName"`
	JobGroup      string    `json:"jobGroup"`
	InvokeTarget  string    `json:"invokeTarget"`
	JobMessage    string    `json:"jobMessage"`
	Status        int       `json:"status"`
	ExceptionInfo string    `json:"exceptionInfo"`
	CreateTime    time.Time `json:"createTime"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
}

type LabelOptionDTO struct {
	Id       int              `json:"id"`
	Label    string           `json:"label"`
	Children []LabelOptionDTO `json:"children"`
}

type MaxwellDataDTO struct {
	Database string                 `json:"database"`
	Xid      string                 `json:"xid"`
	Data     map[string]interface{} `json:"data"`
	Commit   bool                   `json:"commit"`
	Type     string                 `json:"type"`
	Table    string                 `json:"table"`
	Ts       int                    `json:"ts"`
}

type MenuDTO struct {
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Component  string    `json:"component"`
	Icon       string    `json:"icon"`
	CreateTime time.Time `json:"createTime"`
	OrderNum   int       `json:"orderNum"`
	IsDisable  int       `json:"isDisable"`
	IsHidden   int       `json:"isHidden"`
	Children   []MenuDTO `json:"children"`
}

type OperationLogDTO struct {
	Id            int    `json:"id"`
	OptModule     string `json:"optModule"`
	OptUri        string `json:"optUri"`
	OptType       string `json:"optType"`
	OptMethod     string `json:"optMethod"`
	OptDesc       string `json:"optDesc"`
	RequestMethod string `json:"requestMethod"`
	RequestParam  string `json:"requestParam"`
	ResponseData  string `json:"responseData"`
	Nickname      string `json:"nickname"`
	IpAddress     string `json:"ipAddress"`
	IpSource      string `json:"ipSource"`
	CreateTime    string `json:"createTime"`
}

type PageResultDTO struct {
	Records any `json:"records"`
	Count   int `json:"count"`
}

type PhotoAdminDTO struct {
	Id        int    `json:"id"`
	PhotoName string `json:"photoName"`
	PhotoDesc string `json:"photoDesc"`
	PhotoSrc  string `json:"photoSrc"`
}

type PhotoAlbumAdminDTO struct {
	Id         int    `json:"id"`
	AlbumName  string `json:"albumName"`
	AlbumDesc  string `json:"albumDesc"`
	AlbumCover string `json:"albumCover"`
	PhotoCount int    `json:"photoCount"`
	Status     int    `json:"status"`
}

type PhotoAlbumDTO struct {
	Id         int    `json:"id"`
	AlbumName  string `json:"albumName"`
	AlbumDesc  string `json:"albumDesc"`
	AlbumCover string `json:"albumCover"`
}

type PhotoDTO struct {
	PhotoAlbumCover string `json:"photoAlbumCover"`
	PhotoAlbumName  string `json:"photoAlbumName"`
	Photos          any    `json:"photos"`
}

type ReplyDTO = Reply

type ResourceDTO struct {
	Id            int           `json:"id"`
	ResourceName  string        `json:"resourceName"`
	Url           string        `json:"url"`
	RequestMethod string        `json:"requestMethod"`
	IsDisable     int           `json:"isDisable"`
	IsAnonymous   int           `json:"isAnonymous"`
	CreateTime    time.Time     `json:"createTime"`
	Children      []ResourceDTO `json:"children"`
}

type ResourceRoleDTO struct {
	Id            int      `json:"id"`
	Url           string   `json:"url"`
	RequestMethod string   `json:"requestMethod"`
	RoleList      []string `json:"roleList"`
}

type RoleDTO struct {
	Id          int       `json:"id"`
	RoleName    string    `json:"roleName"`
	CreateTime  time.Time `json:"createTime"`
	IsDisable   int       `json:"isDisable"`
	ResourceIds []int     `json:"resourceIds"`
	MenuIds     []int     `json:"menuIds"`
}

type TagAdminDTO = TagAdmin

type TagDTO = Tag

type TalkAdminDTO = TalkAdmin
type TalkDTO = Talk

type TopAndFeaturedArticlesDTO struct {
	TopArticle       *ArticleCardDTO   `json:"topArticle"`
	FeaturedArticles []*ArticleCardDTO `json:"featuredArticles"`
}

type UniqueViewDTO struct {
	Day        string `json:"day"`
	ViewsCount int    `json:"viewsCount"`
}

type UserAdminDTO = UserAdmin

type UserAreaDTO struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type UserDetailsDTO struct {
	Id            int       `json:"id"`
	UserInfoId    int       `json:"userInfoId"`
	Email         string    `json:"email"`
	LoginType     int       `json:"loginType"`
	Username      string    `json:"username"`
	Password      string    `json:"password"`
	Roles         []string  `json:"roles"`
	Nickname      string    `json:"nickname"`
	Avatar        string    `json:"avatar"`
	Intro         string    `json:"intro"`
	Website       string    `json:"website"`
	IsDisable     int       `json:"isDisable"`
	IpAddress     string    `json:"ipAddress"`
	IpSource      string    `json:"ipSource"`
	IsSubscribe   int       `json:"isSubscribe"`
	Browser       string    `json:"browser"`
	Os            string    `json:"os"`
	ExpireTime    time.Time `json:"expireTime"`
	LastLoginTime time.Time `json:"lastLoginTime"`
}

type UserInfoDTO struct {
	Id            int       `json:"id"`
	UserInfoId    int       `json:"userInfoId"`
	Email         string    `json:"email"`
	LoginType     int       `json:"loginType"`
	Username      string    `json:"username"`
	Nickname      string    `json:"nickname"`
	Avatar        string    `json:"avatar"`
	Intro         string    `json:"intro"`
	Website       string    `json:"website"`
	IpAddress     string    `json:"ipAddress"`
	IpSource      string    `json:"ipSource"`
	IsSubscribe   int       `json:"isSubscribe"`
	LastLoginTime time.Time `json:"lastLoginTime"`
	Token         string    `json:"token"`
}

type UserLogoutStatusDTO struct {
	Message string `json:"message"`
}

type UserOnlineDTO struct {
	UserInfoId    int       `json:"userInfoId"`
	Nickname      string    `json:"nickname"`
	Avatar        string    `json:"avatar"`
	IpAddress     string    `json:"ipAddress"`
	IpSource      string    `json:"ipSource"`
	Browser       string    `json:"browser"`
	Os            string    `json:"os"`
	LastLoginTime time.Time `json:"lastLoginTime"`
}

type UserRoleDTO = UserRole

type UserMenuDTO struct {
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	Component string        `json:"component"`
	Icon      string        `json:"icon"`
	Hidden    bool          `json:"hidden"`
	Children  []UserMenuDTO `json:"children"`
}

type WebsiteConfigDTO struct {
	Name              string `json:"name"`
	EnglishName       string `json:"englishName"`
	Author            string `json:"author"`
	AuthorAvatar      string `json:"authorAvatar"`
	AuthorIntro       string `json:"authorIntro"`
	Logo              string `json:"logo"`
	MultiLanguage     int    `json:"multiLanguage"`
	Notice            string `json:"notice"`
	WebsiteCreateTime string `json:"websiteCreateTime"`
	BeianNumber       string `json:"beianNumber"`
	Github            string `json:"github"`
	Gitee             string `json:"gitee"`
	QQ                string `json:"qq"`
	WeChat            string `json:"weChat"`
	Weibo             string `json:"weibo"`
	Csdn              string `json:"csdn"`
	Zhihu             string `json:"zhihu"`
	Juejin            string `json:"juejin"`
	Twitter           string `json:"twitter"`
	Stackoverflow     string `json:"stackoverflow"`
	TouristAvatar     string `json:"touristAvatar"`
	UserAvatar        string `json:"userAvatar"`
	IsCommentReview   int    `json:"isCommentReview"`
	IsEmailNotice     int    `json:"isEmailNotice"`
	IsReward          int    `json:"isReward"`
	WeiXinQRCode      string `json:"weiXinQRCode"`
	AlipayQRCode      string `json:"alipayQRCode"`
}
