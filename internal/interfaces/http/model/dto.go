package model

import (
	"container/list"
	"encoding/json"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"time"
)

type AboutDTO struct {
	Content string `json:"content"`
}

type ArchiveDTO struct {
	Time     string           `json:"time"`
	Articles []ArticleCardDTO `json:"articles"`
}

type ArticleAdminDTO = port.ArticleAdmin
type ArticleCardDTO = port.ArticleCard
type ArticleAdminViewDTO = port.ArticleAdminView
type ArticleDTO = port.Article

type ArticleRankDTO struct {
	ArticleTitle string `json:"articleTitle"`
	ViewsCount   int    `json:"viewsCount"`
}

type ArticleSearchDTO = port.ArticleSearch
type ArticleStatisticsDTO = port.ArticleStatistics

type StellarBeaconAdminInfoDTO struct {
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

type StellarBeaconBackInfoDTO struct {
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

type DashboardOverviewDTO struct {
	TotalViews   int `json:"totalViews"`
	TodayViews   int `json:"todayViews"`
	MonthViews   int `json:"monthViews"`
	UserCount    int `json:"userCount"`
	ArticleCount int `json:"articleCount"`
	MessageCount int `json:"messageCount"`
}

type DashboardTrendDTO struct {
	Period string `json:"period"`
	Views  int    `json:"views"`
}

type DashboardRegionDTO struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Code  string `json:"code"`
	Value int64  `json:"value"`
}

type DashboardDistributionDTO struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type DashboardArticleRankDTO struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Views int    `json:"views"`
}

type DashboardAnalyticsDTO struct {
	Range       string                     `json:"range"`
	Unit        string                     `json:"unit"`
	Overview    DashboardOverviewDTO       `json:"overview"`
	Trend       []DashboardTrendDTO        `json:"trend"`
	Regions     []DashboardRegionDTO       `json:"regions"`
	Categories  []DashboardDistributionDTO `json:"categories"`
	Tags        []DashboardDistributionDTO `json:"tags"`
	ArticleRank []DashboardArticleRankDTO  `json:"articleRank"`
	Growth      DashboardGrowthDTO         `json:"growth"`
	GeneratedAt time.Time                  `json:"generatedAt"`
}

type GrowthSubscriberStatsDTO struct {
	Total            int64   `json:"total"`
	Active           int64   `json:"active"`
	Pending          int64   `json:"pending"`
	Unsubscribed     int64   `json:"unsubscribed"`
	ConfirmationRate float64 `json:"confirmationRate"`
}

type GrowthDeliveryStatsDTO struct {
	Queued      int64   `json:"queued"`
	Sending     int64   `json:"sending"`
	Sent        int64   `json:"sent"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"successRate"`
}

type DashboardGrowthTrendDTO struct {
	Period            string `json:"period"`
	ShareClicks       int64  `json:"shareClicks"`
	SubscribeStarts   int64  `json:"subscribeStarts"`
	SubscribeConfirms int64  `json:"subscribeConfirms"`
	Unsubscribes      int64  `json:"unsubscribes"`
	DeliverySent      int64  `json:"deliverySent"`
	DeliveryFailed    int64  `json:"deliveryFailed"`
}

type DashboardGrowthDTO struct {
	Subscribers GrowthSubscriberStatsDTO  `json:"subscribers"`
	Deliveries  GrowthDeliveryStatsDTO    `json:"deliveries"`
	Trend       []DashboardGrowthTrendDTO `json:"trend"`
}

type SMTPHealthDTO struct {
	Configured bool      `json:"configured"`
	Reachable  bool      `json:"reachable"`
	Host       string    `json:"host"`
	Port       int       `json:"port"`
	TLS        bool      `json:"tls"`
	Auth       bool      `json:"auth"`
	CheckedAt  time.Time `json:"checkedAt"`
	Message    string    `json:"message"`
}

type NewsletterHealthDTO struct {
	Subscribers GrowthSubscriberStatsDTO `json:"subscribers"`
	Deliveries  GrowthDeliveryStatsDTO   `json:"deliveries"`
	SMTP        SMTPHealthDTO            `json:"smtp"`
	GeneratedAt time.Time                `json:"generatedAt"`
}

type MediaAssetDTO struct {
	Key          string    `json:"key"`
	URL          string    `json:"url"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType"`
	LastModified time.Time `json:"lastModified"`
	Deletable    bool      `json:"deletable"`
}

type StellarBeaconHomeInfoDTO struct {
	ArticleCount    int64            `json:"articleCount"`
	TalkCount       int64            `json:"talkCount"`
	CategoryCount   int64            `json:"categoryCount"`
	TagCount        int64            `json:"tagCount"`
	WebsiteConfigDT WebsiteConfigDTO `json:"websiteConfigDTO"`
	ViewCount       int              `json:"viewCount"`
}

type CategoryAdminDTO = port.CategoryAdmin
type CategoryDTO = port.Category
type CategoryOptionDTO = port.CategoryOption

type CommentAdminDTO = port.CommentAdmin
type CommentCountDTO = port.CommentCount
type CommentDTO = port.Comment

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
	// Records and Count are retained for application-service compatibility.
	// The HTTP representation is emitted by MarshalJSON using the new names.
	Records  any `json:"-"`
	Count    int `json:"-"`
	Page     int `json:"-"`
	PageSize int `json:"-"`
}

func (p PageResultDTO) MarshalJSON() ([]byte, error) {
	items := pageItems(p.Records)
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize < 0 {
		pageSize = 0
	}
	if pageSize == 0 {
		if values, ok := items.([]interface{}); ok {
			pageSize = len(values)
		}
	}
	return json.Marshal(struct {
		Items    interface{} `json:"items"`
		Total    int         `json:"total"`
		Page     int         `json:"page"`
		PageSize int         `json:"pageSize"`
	}{Items: items, Total: p.Count, Page: page, PageSize: pageSize})
}

func pageItems(value any) any {
	switch records := value.(type) {
	case nil:
		return []interface{}{}
	case *list.List:
		if records == nil {
			return []interface{}{}
		}
		items := make([]interface{}, 0, records.Len())
		for element := records.Front(); element != nil; element = element.Next() {
			items = append(items, element.Value)
		}
		return items
	case list.List:
		items := make([]interface{}, 0, records.Len())
		for element := records.Front(); element != nil; element = element.Next() {
			items = append(items, element.Value)
		}
		return items
	default:
		return records
	}
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

type ReplyDTO = port.Reply

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

type TagAdminDTO = port.TagAdmin

type TagDTO = port.Tag

type TalkAdminDTO = port.TalkAdmin
type TalkDTO = port.Talk

type TopAndFeaturedArticlesDTO struct {
	TopArticle       *ArticleCardDTO   `json:"topArticle"`
	FeaturedArticles []*ArticleCardDTO `json:"featuredArticles"`
}

type UniqueViewDTO struct {
	Day        string `json:"day"`
	ViewsCount int    `json:"viewsCount"`
}

type UserAdminDTO = port.UserAdmin

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
	NotifyComment int       `json:"notifyComment"`
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

type UserRoleDTO = port.UserRole

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

// ReactionToggleVO is the reader-interaction write payload. Active carries the
// desired state rather than a blind toggle so retries stay idempotent.
type ReactionToggleVO struct {
	ArticleId int    `json:"articleId" form:"articleId"`
	Reaction  string `json:"reaction" form:"reaction"`
	Active    bool   `json:"active" form:"active"`
}

// ReactionToggleDTO reports the resulting state and the refreshed totals.
type ReactionToggleDTO struct {
	Active        bool `json:"active"`
	LikeCount     int  `json:"likeCount"`
	FavoriteCount int  `json:"favoriteCount"`
}

// ReactionStateDTO is one row of the per-account state lookup.
type ReactionStateDTO struct {
	ArticleId int  `json:"articleId"`
	Like      bool `json:"like"`
	Favorite  bool `json:"favorite"`
}

// CommentNoticeVO toggles the comment notification preference of the
// authenticated account.
type CommentNoticeVO struct {
	NotifyComment int `json:"notifyComment" form:"notifyComment"`
}
