package model

import (
	"benetnasch/app/domain/entity"
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

type ArticleAdminDTO struct {
	Id           int       `json:"id"`
	ArticleCover string    `json:"articleCover"`
	ArticleTitle string    `json:"articleTitle"`
	IsTop        int       `json:"isTop"`
	IsFeatured   int       `json:"isFeatured"`
	IsDelete     int       `json:"isDelete"`
	Status       int       `json:"status"`
	Type         int       `json:"type"`
	CreateTime   time.Time `json:"createTime"`
	CategoryName string    `json:"categoryName"`
	ViewsCount   int       `json:"viewsCount"`
	TagDTOs      []TagDTO  `json:"tagDTOs"`
}

type ArticleCardDTO struct {
	Id             int              `json:"id"`
	ArticleCover   string           `json:"articleCover"`
	ArticleTitle   string           `json:"articleTitle"`
	ArticleContent string           `json:"articleContent"`
	IsTop          int              `json:"isTop"`
	IsFeatured     int              `json:"isFeatured"`
	CategoryName   string           `json:"categoryName"`
	Status         int              `json:"status"`
	CreateTime     time.Time        `json:"createTime"`
	UpdateTime     time.Time        `json:"updateTime"`
	Author         entity.TUserInfo `json:"author" xorm:"extends"`
	Tags           any              `json:"tags" xorm:"extends"`
}

type ArticleAdminViewDTO struct {
	Id             int    `json:"id"`
	ArticleCover   string `json:"articleCover"`
	ArticleTitle   string `json:"articleTitle"`
	ArticleContent string `json:"articleContent"`
	IsTop          int    `json:"isTop"`
	IsFeatured     int    `json:"isFeatured"`
	CategoryName   string `json:"categoryName"`
	TagNames       any    `json:"tagNames"`
	Status         int    `json:"status"`
	Type           int    `json:"type"`
	Password       string `json:"password"`
	OriginalUrl    string `json:"originalUrl"`
}

type ArticleDTO struct {
	Id              int              `json:"id"`
	ArticleCover    string           `json:"articleCover"`
	ArticleTitle    string           `json:"articleTitle"`
	ArticleContent  string           `json:"articleContent"`
	IsTop           int              `json:"isTop"`
	IsFeatured      int              `json:"isFeatured"`
	CategoryName    string           `json:"categoryName"`
	Status          int              `json:"status"`
	CreateTime      time.Time        `json:"createTime"`
	UpdateTime      time.Time        `json:"updateTime"`
	Author          entity.TUserInfo `json:"author" xorm:"extends"`
	Type            int              `json:"type"`
	OriginalUrl     string           `json:"originalUrl"`
	IsDelete        int              `json:"isDelete"`
	ViewCount       int              `json:"viewCount"`
	PreArticleCard  ArticleCardDTO   `json:"preArticleCard"`
	NextArticleCard ArticleCardDTO   `json:"nextArticleCard"`
	Tags            any              `json:"tags"`
}

type ArticleRankDTO struct {
	ArticleTitle string `json:"articleTitle"`
	ViewsCount   int    `json:"viewsCount"`
}

type ArticleSearchDTO struct {
	Id             int    `json:"id"`
	ArticleTitle   string `json:"articleTitle"`
	ArticleContent string `json:"articleContent"`
	IsDelete       int    `json:"isDelete"`
	Status         int    `json:"status"`
}

type ArticleStatisticsDTO struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
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

type CategoryAdminDTO struct {
	Id           int       `json:"id"`
	CategoryName string    `json:"categoryName"`
	ArticleCount string    `json:"articleCount"`
	CreateTime   time.Time `json:"createTime"`
}

type CategoryDTO struct {
	Id           int    `json:"id"`
	CategoryName string `json:"categoryName"`
	ArticleCount int    `json:"articleCount"`
}

type CategoryOptionDTO struct {
	Id           int    `json:"id"`
	CategoryName string `json:"categoryName"`
}

type CommentAdminDTO struct {
	Id             int       `json:"id"`
	Avatar         string    `json:"avatar"`
	Nickname       string    `json:"nickname"`
	ReplyNickname  string    `json:"replyNickname"`
	ArticleTitle   string    `json:"articleTitle"`
	CommentContent string    `json:"commentContent"`
	Type           int       `json:"type"`
	IsReview       int       `json:"isReview"`
	CreateTime     time.Time `json:"createTime"`
}

type CommentCountDTO struct {
	Id           int `json:"id"`
	CommentCount int `json:"commentCount"`
}

type CommentDTO struct {
	Id             int        `json:"id"`
	UserId         int        `json:"userId"`
	Nickname       string     `json:"nickname"`
	Avatar         string     `json:"avatar"`
	Website        string     `json:"website"`
	CommentContent string     `json:"commentContent"`
	CreateTime     time.Time  `json:"createTime"`
	ReplyDTOs      []ReplyDTO `json:"replyDTOs"`
}

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

type QQTokenDTO struct {
	Openid    string `json:"openid"`
	Client_id string `json:"client_Id"`
}

type QQUserInfoDTO struct {
	Nickname       string `json:"nickname"`
	Figureurl_qq_1 string `json:"figureurl_qq_1"`
}

type ReplyDTO struct {
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
}

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

type SocialTokenDTO struct {
	OpenId      string `json:"openId"`
	AccessToken string `json:"accessToken"`
	LoginType   int    `json:"loginType"`
}

type SocialUserInfoDTO struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type TagAdminDTO struct {
	Id           int       `json:"id"`
	TagName      string    `json:"tagName"`
	ArticleCount int       `json:"articleCount"`
	CreateTime   time.Time `json:"createTime"`
}

type TagDTO struct {
	Id      int    `json:"id"`
	TagName string `json:"tagName"`
	Count   int    `json:"count"`
}

type TalkAdminDTO struct {
	Id         int       `json:"id"`
	Nickname   string    `json:"nickname"`
	Avatar     string    `json:"avatar"`
	Content    string    `json:"content"`
	Images     string    `json:"images"`
	Imgs       []string  `json:"imgs"`
	IsTop      int       `json:"isTop"`
	Status     int       `json:"status"`
	CreateTime time.Time `json:"createTime"`
}

type TalkDTO struct {
	Id           int       `json:"id"`
	Nickname     string    `json:"nickName"`
	Avatar       string    `json:"avatar"`
	Content      string    `json:"content"`
	Images       string    `json:"images"`
	Imgs         []string  `json:"imgs"`
	IsTop        int       `json:"isTop"`
	CommentCount int       `json:"commentCount"`
	CreateTime   time.Time `json:"createTime"`
}

type TopAndFeaturedArticlesDTO struct {
	TopArticle       *ArticleCardDTO   `json:"topArticle"`
	FeaturedArticles []*ArticleCardDTO `json:"featuredArticles"`
}

type UniqueViewDTO struct {
	Day        string `json:"day"`
	ViewsCount int    `json:"viewsCount"`
}

type UserAdminDTO struct {
	Id            int           `json:"id"`
	UserInfoId    int           `json:"userInfoId" xorm:"not null comment('用户信息id') INTEGER"`
	Avatar        string        `json:"avatar"`
	Nickname      string        `json:"nickname"`
	LoginType     int           `json:"loginType"`
	IpAddress     string        `json:"ipAddress"`
	IpSource      string        `json:"ipSource"`
	CreateTime    time.Time     `json:"createTime"`
	LastLoginTime time.Time     `json:"lastLoginTime"`
	IsDisable     int           `json:"isDisable"`
	Status        int           `json:"status"`
	Roles         []UserRoleDTO `json:"roles"`
}

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

type UserRoleDTO struct {
	Id       int    `json:"id"`
	RoleName string `json:"roleName"`
}

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
	QQLogin           int    `json:"qqLogin"`
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
