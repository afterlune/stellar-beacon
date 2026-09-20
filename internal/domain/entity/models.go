package entity

import (
	"time"
)

type TAbout struct {
	Id         int       `xorm:"autoincr not null pk INTEGER"`
	Content    string    `xorm:"comment('内容') TEXT"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME"`
}

type TArticle struct {
	Id             int    `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	UserId         int    `xorm:"not null comment('作者') INTEGER" json:"userId"`
	CategoryId     int    `xorm:"comment('文章分类') INTEGER" json:"categoryId"`
	ArticleCover   string `xorm:"comment('文章缩略图') VARCHAR(1024)" json:"articleCover"`
	ArticleTitle   string `xorm:"not null comment('标题') VARCHAR(50)" json:"articleTitle"`
	ArticleContent string `xorm:"not null comment('内容') TEXT" json:"articleContent"`
	// The column must be spelled out: xorm's snake mapper turns the `HTML`
	// suffix into `article_content_h_t_m_l`, which does not exist.
	ArticleContentHTML string    `xorm:"article_content_html comment('富文本 HTML 内容') TEXT" json:"articleContentHtml,omitempty"`
	SeriesId           int       `xorm:"series_id comment('所属系列') INTEGER" json:"seriesId"`
	SeriesOrder        int       `xorm:"series_order not null default 0 comment('系列内序号') INTEGER" json:"seriesOrder"`
	ScheduledAt        time.Time `xorm:"scheduled_at comment('定时发布时间') DATETIME" json:"scheduledAt,omitempty"`
	IsTop              int       `xorm:"not null comment('是否置顶 0否 1是') SMALLINT" json:"isTop"`
	IsFeatured         int       `xorm:"not null comment('是否推荐 0否 1是') SMALLINT" json:"isFeatured"`
	IsDelete           int       `xorm:"not null comment('是否删除  0否 1是') SMALLINT" json:"isDelete"`
	Status             int       `xorm:"not null comment('状态值 1公开 2私密 3草稿 4定时') SMALLINT" json:"status"`
	ModerationStatus   string    `xorm:"moderation_status not null default 'visible' comment('审核状态 visible/hidden') VARCHAR(16)" json:"moderationStatus"`
	ModerationReason   string    `xorm:"moderation_reason comment('审核下架原因') VARCHAR(255)" json:"moderationReason,omitempty"`
	ModeratedBy        int       `xorm:"moderated_by default 0 comment('审核人 user_info id') INTEGER" json:"moderatedBy,omitempty"`
	ModeratedAt        time.Time `xorm:"moderated_at comment('审核时间') DATETIME" json:"moderatedAt,omitempty"`
	Type               int       `xorm:"not null comment('文章类型 1原创 2转载 3翻译') SMALLINT" json:"type"`
	Password           string    `xorm:"comment('访问密码') VARCHAR(255)" json:"password"`
	OriginalUrl        string    `xorm:"comment('原文链接') VARCHAR(255)" json:"originalUrl"`
	CreateTime         time.Time `xorm:"created not null comment('发表时间') DATETIME" json:"createTime"`
	UpdateTime         time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

// TArticlePublishRecord is the durable hand-off between the scheduled
// publisher and subscriber notification enqueueing. The article status flip
// and this row are committed in the same transaction.
type TArticlePublishRecord struct {
	Id                   int       `xorm:"autoincr not null pk unique BIGINT" json:"id"`
	ArticleId            int       `xorm:"article_id not null unique(article_schedule) index INTEGER" json:"articleId"`
	UserId               int       `xorm:"user_id not null index INTEGER" json:"userId"`
	ScheduledAt          time.Time `xorm:"scheduled_at not null unique(article_schedule) DATETIME" json:"scheduledAt"`
	PublishedAt          time.Time `xorm:"published_at not null DATETIME" json:"publishedAt"`
	NotificationState    string    `xorm:"notification_state not null default 'pending' VARCHAR(20)" json:"notificationState"`
	NotificationAttempts int       `xorm:"notification_attempts not null default 0 INTEGER" json:"notificationAttempts"`
	NextRetryAt          time.Time `xorm:"next_retry_at DATETIME" json:"nextRetryAt,omitempty"`
	LastError            string    `xorm:"last_error not null default '' TEXT" json:"lastError,omitempty"`
	CreateTime           time.Time `xorm:"created not null DATETIME" json:"createTime"`
	UpdateTime           time.Time `xorm:"updated not null DATETIME" json:"updateTime"`
}

type TContentOperationAudit struct {
	Id               int       `xorm:"autoincr not null pk unique BIGINT" json:"id"`
	OperatorId       int       `xorm:"operator_id not null index INTEGER" json:"operatorId"`
	OperatorNickname string    `xorm:"operator_nickname not null VARCHAR(64)" json:"operatorNickname"`
	ContentType      string    `xorm:"content_type not null VARCHAR(16)" json:"contentType"`
	Operation        string    `xorm:"operation not null VARCHAR(32)" json:"operation"`
	TargetMode       string    `xorm:"target_mode not null VARCHAR(16)" json:"targetMode"`
	FilterSnapshot   string    `xorm:"filter_snapshot not null default '{}' JSONB" json:"filterSnapshot"`
	SnapshotMaxId    int       `xorm:"snapshot_max_id not null default 0 INTEGER" json:"snapshotMaxId"`
	RequestedCount   int       `xorm:"requested_count not null default 0 INTEGER" json:"requestedCount"`
	AffectedCount    int       `xorm:"affected_count not null default 0 INTEGER" json:"affectedCount"`
	Result           string    `xorm:"result not null VARCHAR(16)" json:"result"`
	ErrorMessage     string    `xorm:"error_message not null default '' TEXT" json:"errorMessage,omitempty"`
	IpAddress        string    `xorm:"ip_address not null default '' VARCHAR(255)" json:"ipAddress"`
	IpSource         string    `xorm:"ip_source not null default '' VARCHAR(255)" json:"ipSource"`
	CreateTime       time.Time `xorm:"created not null created_at DATETIME" json:"createTime"`
}

type TContentOperationAuditItem struct {
	Id             int    `xorm:"autoincr not null pk unique BIGINT" json:"id"`
	AuditId        int    `xorm:"audit_id not null index BIGINT" json:"auditId"`
	ContentId      int    `xorm:"content_id not null INTEGER" json:"contentId"`
	Title          string `xorm:"title not null default '' VARCHAR(255)" json:"title"`
	PreviousStatus int    `xorm:"previous_status not null SMALLINT" json:"previousStatus"`
	NextStatus     int    `xorm:"next_status not null SMALLINT" json:"nextStatus"`
	Result         string `xorm:"result not null VARCHAR(16)" json:"result"`
}

// TSeries groups articles into an ordered collection. The relation lives on
// t_article.series_id so an article belongs to at most one series.
type TSeries struct {
	Id               int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	UserId           int       `xorm:"user_id not null index comment('作者') INTEGER" json:"userId"`
	SeriesName       string    `xorm:"series_name not null comment('系列名') VARCHAR(50)" json:"seriesName"`
	SeriesDesc       string    `xorm:"series_desc comment('系列描述') VARCHAR(255)" json:"seriesDesc"`
	Cover            string    `xorm:"comment('系列封面') VARCHAR(1024)" json:"cover"`
	Status           int       `xorm:"not null default 1 comment('状态值 1公开 2私密 3草稿') SMALLINT" json:"status"`
	ModerationStatus string    `xorm:"moderation_status not null default 'visible' comment('审核状态 visible/hidden') VARCHAR(16)" json:"moderationStatus"`
	ModerationReason string    `xorm:"moderation_reason comment('审核下架原因') VARCHAR(255)" json:"moderationReason,omitempty"`
	ModeratedBy      int       `xorm:"moderated_by default 0 comment('审核人 user_info id') INTEGER" json:"moderatedBy,omitempty"`
	ModeratedAt      time.Time `xorm:"moderated_at comment('审核时间') DATETIME" json:"moderatedAt,omitempty"`
	IsDelete         int       `xorm:"not null default 0 comment('是否删除 0否 1是') SMALLINT" json:"isDelete"`
	CreateTime       time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime       time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TArticleTag struct {
	Id        int `xorm:"autoincr not null pk unique INTEGER"`
	ArticleId int `xorm:"not null comment('文章id') index INTEGER"`
	TagId     int `xorm:"not null comment('标签id') index INTEGER"`
}

type TCategory struct {
	Id           int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	UserId       int       `xorm:"user_id not null index comment('所属用户') INTEGER" json:"userId"`
	CategoryName string    `xorm:"not null comment('分类名') VARCHAR(20)" json:"categoryName"`
	CreateTime   time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime   time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TComment struct {
	Id             int       `xorm:"autoincr not null pk comment('主键') unique INTEGER" json:"id"`
	UserId         int       `xorm:"not null comment('评论用户Id') index INTEGER" json:"userId"`
	TopicId        int       `xorm:"comment('评论主题id') INTEGER" json:"topicId"`
	CommentContent string    `xorm:"not null comment('评论内容') TEXT" json:"commentContent"`
	ReplyUserId    int       `xorm:"comment('回复用户id') INTEGER" json:"replyUserId"`
	ParentId       int       `xorm:"comment('父评论id') index INTEGER" json:"parentId"`
	Type           int       `xorm:"not null comment('评论类型 1.文章 2.留言 3.关于我 4.友链 5.说说') SMALLINT" json:"type"`
	IsDelete       int       `xorm:"not null comment('是否删除  0否 1是') SMALLINT" json:"isDelete"`
	IsReview       int       `xorm:"not null comment('是否审核') SMALLINT" json:"isReview"`
	CreateTime     time.Time `xorm:"created not null comment('评论时间') DATETIME" json:"createTime"`
	UpdateTime     time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TExceptionLog struct {
	Id            int       `xorm:"autoincr not null pk unique INTEGER"`
	OptUri        string    `xorm:"not null comment('请求接口') VARCHAR(255)"`
	OptMethod     string    `xorm:"not null comment('请求方式') VARCHAR(255)"`
	RequestMethod string    `xorm:"comment('请求方式') VARCHAR(255)"`
	RequestParam  string    `xorm:"comment('请求参数') VARCHAR(2000)"`
	OptDesc       string    `xorm:"comment('操作描述') VARCHAR(255)"`
	ExceptionInfo string    `xorm:"comment('异常信息') TEXT"`
	IpAddress     string    `xorm:"comment('ip') VARCHAR(255)"`
	IpSource      string    `xorm:"comment('ip来源') VARCHAR(255)"`
	CreateTime    time.Time `xorm:"created not null comment('操作时间') DATETIME"`
}

type TFriendLink struct {
	Id             int       `xorm:"autoincr not null pk unique INTEGER"`
	LinkName       string    `xorm:"not null comment('链接名') index VARCHAR(20)"`
	LinkAvatar     string    `xorm:"not null comment('链接头像') VARCHAR(255)"`
	LinkAddress    string    `xorm:"not null comment('链接地址') VARCHAR(255)"`
	LinkIntro      string    `xorm:"not null comment('链接介绍') VARCHAR(100)"`
	Status         int       `xorm:"status not null default 1 comment('状态 0待审 1通过 2拒绝') SMALLINT" json:"status"`
	ApplicantEmail string    `xorm:"applicant_email comment('申请人邮箱') VARCHAR(254)" json:"applicantEmail"`
	AuditTime      time.Time `xorm:"audit_time comment('审核时间') DATETIME" json:"auditTime"`
	CreateTime     time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime     time.Time `xorm:"updated comment('更新时间') DATETIME"`
}

type TJob struct {
	Id             int       `xorm:"autoincr not null pk comment('任务ID') unique(_copy_17) INTEGER"`
	JobName        string    `xorm:"not null pk comment('任务名称') unique(_copy_17) VARCHAR(64)"`
	JobGroup       string    `xorm:"not null pk comment('任务组名') unique(_copy_17) VARCHAR(64)"`
	InvokeTarget   string    `xorm:"not null comment('调用目标字符串') VARCHAR(500)"`
	CronExpression string    `xorm:"comment('cron执行表达式') VARCHAR(255)"`
	MisfirePolicy  int       `xorm:"comment('计划执行错误策略（1立即执行 2执行一次 3放弃执行）') SMALLINT"`
	Concurrent     int       `xorm:"comment('是否并发执行（0禁止 1允许）') SMALLINT"`
	Status         int       `xorm:"comment('状态（0暂停 1正常）') SMALLINT"`
	CreateTime     time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime     time.Time `xorm:"updated comment('更新时间') DATETIME"`
	Remark         string    `xorm:"comment('备注信息') VARCHAR(500)"`
}

type TJobLog struct {
	Id            int       `xorm:"autoincr not null pk comment('任务日志ID') unique INTEGER"`
	JobId         int       `xorm:"not null comment('任务ID') INTEGER"`
	JobName       string    `xorm:"not null comment('任务名称') VARCHAR(64)"`
	JobGroup      string    `xorm:"not null comment('任务组名') VARCHAR(64)"`
	InvokeTarget  string    `xorm:"not null comment('调用目标字符串') VARCHAR(500)"`
	JobMessage    string    `xorm:"comment('日志信息') VARCHAR(500)"`
	Status        int       `xorm:"comment('执行状态（0正常 1失败）') SMALLINT"`
	ExceptionInfo string    `xorm:"comment('异常信息') VARCHAR(2000)"`
	CreateTime    time.Time `xorm:"created comment('创建时间') DATETIME"`
	StartTime     time.Time `xorm:"comment('开始时间') DATETIME"`
	EndTime       time.Time `xorm:"comment('结束时间') DATETIME"`
}

type TMenu struct {
	Id         int       `xorm:"autoincr not null pk comment('主键') unique INTEGER" json:"id"`
	Name       string    `xorm:"not null comment('菜单名') VARCHAR(20)" json:"name"`
	Path       string    `xorm:"not null comment('菜单路径') VARCHAR(50)" json:"path"`
	Component  string    `xorm:"not null comment('组件') VARCHAR(50)" json:"component"`
	Icon       string    `xorm:"not null comment('菜单icon') VARCHAR(50)" json:"icon"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
	OrderNum   int       `xorm:"not null comment('排序') SMALLINT" json:"orderNum"`
	ParentId   int       `xorm:"comment('父id') INTEGER" json:"parentId"`
	IsHidden   int       `xorm:"not null comment('是否隐藏  0否1是') SMALLINT" json:"isHidden"`
}

type TOperationLog struct {
	Id            int       `xorm:"autoincr not null pk comment('主键id')"`
	OptModule     string    `xorm:"not null comment('操作模块') VARCHAR(20)"`
	OptType       string    `xorm:"not null comment('操作类型') VARCHAR(20)"`
	OptUri        string    `xorm:"not null comment('操作url') VARCHAR(255)"`
	OptMethod     string    `xorm:"not null comment('操作方法') VARCHAR(255)"`
	OptDesc       string    `xorm:"not null comment('操作描述') VARCHAR(255)"`
	RequestParam  string    `xorm:"not null comment('请求参数') TEXT"`
	RequestMethod string    `xorm:"not null comment('请求方式') VARCHAR(20)"`
	ResponseData  string    `xorm:"not null comment('返回数据') TEXT"`
	UserId        int       `xorm:"not null comment('用户id') INTEGER"`
	Nickname      string    `xorm:"not null comment('用户昵称') VARCHAR(50)"`
	IpAddress     string    `xorm:"not null comment('操作ip') VARCHAR(255)"`
	IpSource      string    `xorm:"not null comment('操作地址') VARCHAR(255)"`
	CreateTime    time.Time `xorm:"created not null comment('创建时间')"`
	UpdateTime    time.Time `xorm:"updated comment('更新时间')"`
}

type TPhoto struct {
	Id         int       `xorm:"autoincr not null pk comment('主键') unique INTEGER"`
	AlbumId    int       `xorm:"not null comment('相册id') INTEGER"`
	PhotoName  string    `xorm:"not null comment('照片名') VARCHAR(20)"`
	PhotoDesc  string    `xorm:"comment('照片描述') VARCHAR(50)"`
	PhotoSrc   string    `xorm:"not null comment('照片地址') VARCHAR(255)"`
	IsDelete   int       `xorm:"not null comment('是否删除') SMALLINT"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME"`
}

type TPhotoAlbum struct {
	Id         int       `xorm:"autoincr not null pk comment('主键') unique INTEGER"`
	AlbumName  string    `xorm:"not null comment('相册名') VARCHAR(20)"`
	AlbumDesc  string    `xorm:"not null comment('相册描述') VARCHAR(50)"`
	AlbumCover string    `xorm:"not null comment('相册封面') VARCHAR(255)"`
	IsDelete   int       `xorm:"not null comment('是否删除') SMALLINT"`
	Status     int       `xorm:"not null comment('状态值 1公开 2私密') SMALLINT"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME"`
}

type TResource struct {
	Id            int       `xorm:"autoincr not null pk comment('主键') unique INTEGER"`
	ResourceName  string    `xorm:"not null comment('资源名') VARCHAR(50)"`
	Url           string    `xorm:"comment('权限路径') VARCHAR(255)"`
	RequestMethod string    `xorm:"comment('请求方式') VARCHAR(10)"`
	ParentId      int       `xorm:"comment('父模块id') INTEGER"`
	IsAnonymous   int       `xorm:"not null comment('是否匿名访问 0否 1是') SMALLINT"`
	CreateTime    time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime    time.Time `xorm:"updated comment('修改时间') DATETIME"`
}

type TRole struct {
	Id         int       `xorm:"autoincr not null pk comment('主键id') unique INTEGER"`
	RoleName   string    `xorm:"not null comment('角色名') VARCHAR(20)"`
	IsDisable  int       `xorm:"not null comment('是否禁用  0否 1是') SMALLINT"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME"`
}

type TRoleMenu struct {
	Id     int `xorm:"autoincr not null pk comment('主键') unique INTEGER"`
	RoleId int `xorm:"comment('角色id') INTEGER"`
	MenuId int `xorm:"comment('菜单id') INTEGER"`
}

type TRoleResource struct {
	Id         int `xorm:"autoincr not null pk unique INTEGER"`
	RoleId     int `xorm:"comment('角色id') INTEGER"`
	ResourceId int `xorm:"comment('权限id') INTEGER"`
}

type TTag struct {
	Id         int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	UserId     int       `xorm:"user_id not null index comment('所属用户') INTEGER" json:"userId"`
	TagName    string    `xorm:"not null comment('标签名') VARCHAR(20)" json:"tagName"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TTalk struct {
	Id               int       `xorm:"autoincr not null pk comment('说说id') unique INTEGER" json:"id"`
	UserId           int       `xorm:"not null comment('用户id') INTEGER" json:"userId"`
	Content          string    `xorm:"not null comment('说说内容') VARCHAR(2000)" json:"content"`
	Images           string    `xorm:"comment('图片') VARCHAR(2500)" json:"images"`
	IsTop            int       `xorm:"not null comment('是否置顶') SMALLINT" json:"isTop"`
	Status           int       `xorm:"not null comment('状态 1.公开 2.私密 3.草稿') SMALLINT" json:"status"`
	ModerationStatus string    `xorm:"moderation_status not null default 'visible' comment('审核状态 visible/hidden') VARCHAR(16)" json:"moderationStatus"`
	ModerationReason string    `xorm:"moderation_reason comment('审核下架原因') VARCHAR(255)" json:"moderationReason,omitempty"`
	ModeratedBy      int       `xorm:"moderated_by default 0 comment('审核人 user_info id') INTEGER" json:"moderatedBy,omitempty"`
	ModeratedAt      time.Time `xorm:"moderated_at comment('审核时间') DATETIME" json:"moderatedAt,omitempty"`
	CreateTime       time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime       time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TUniqueView struct {
	Id         int       `xorm:"autoincr not null pk unique INTEGER"`
	ViewsCount int       `xorm:"not null comment('访问量') INTEGER"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME"`
}

// TArticleDailyMetric stores only aggregate article performance counters.
// Reader identity stays in Redis HyperLogLogs and is never persisted here.
type TArticleDailyMetric struct {
	Id                 int       `xorm:"autoincr not null pk unique INTEGER"`
	ArticleId          int       `xorm:"article_id not null unique(article_day) index INTEGER"`
	MetricDate         time.Time `xorm:"metric_date not null unique(article_day) index DATE"`
	Views              int64     `xorm:"not null default 0 BIGINT"`
	UniqueReaders      int64     `xorm:"not null default 0 BIGINT"`
	EffectiveSessions  int64     `xorm:"not null default 0 BIGINT"`
	TotalActiveMs      int64     `xorm:"not null default 0 BIGINT"`
	CompletedSessions  int64     `xorm:"not null default 0 BIGINT"`
	SeriesImpressions  int64     `xorm:"not null default 0 BIGINT"`
	SeriesClicks       int64     `xorm:"not null default 0 BIGINT"`
	RelatedImpressions int64     `xorm:"not null default 0 BIGINT"`
	RelatedClicks      int64     `xorm:"not null default 0 BIGINT"`
	CreateTime         time.Time `xorm:"created not null DATETIME"`
	UpdateTime         time.Time `xorm:"updated not null DATETIME"`
}

// TArticleContinuationTarget aggregates clicks on one attributed continuation
// target. The target id is intentionally not a foreign key because it can
// reference either t_article or t_series.
type TArticleContinuationTarget struct {
	Id              int       `xorm:"autoincr not null pk unique INTEGER"`
	SourceArticleId int       `xorm:"source_article_id not null unique(source_target_day) index INTEGER"`
	TargetType      string    `xorm:"target_type not null unique(source_target_day) VARCHAR(16)"`
	TargetId        int       `xorm:"target_id not null unique(source_target_day) index INTEGER"`
	Placement       string    `xorm:"placement not null unique(source_target_day) VARCHAR(24)"`
	Position        int       `xorm:"position not null default 0 unique(source_target_day) SMALLINT"`
	MetricDate      time.Time `xorm:"metric_date not null unique(source_target_day) index DATE"`
	Clicks          int64     `xorm:"not null default 0 BIGINT"`
	CreateTime      time.Time `xorm:"created not null DATETIME"`
	UpdateTime      time.Time `xorm:"updated not null DATETIME"`
}
type TUserAuth struct {
	Id            int       `xorm:"autoincr not null pk unique INTEGER"`
	UserInfoId    int       `xorm:"not null comment('用户信息id') INTEGER"`
	Username      string    `xorm:"not null comment('用户名') unique VARCHAR(50)"`
	Password      string    `xorm:"not null comment('密码') VARCHAR(100)"`
	LoginType     int       `xorm:"not null comment('登录类型') SMALLINT"`
	IpAddress     string    `xorm:"comment('用户登录ip') VARCHAR(50)"`
	IpSource      string    `xorm:"comment('ip来源') VARCHAR(50)"`
	CreateTime    time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime    time.Time `xorm:"updated comment('更新时间') DATETIME"`
	LastLoginTime time.Time `xorm:"comment('上次登录时间') DATETIME"`
}

type TUserInfo struct {
	Id            int       `xorm:"autoincr not null pk comment('用户ID') unique INTEGER" json:"id"`
	Handle        string    `xorm:"handle not null unique comment('公开唯一标识') VARCHAR(40)" json:"handle"`
	Email         string    `xorm:"comment('邮箱号') VARCHAR(50)" json:"email"`
	Nickname      string    `xorm:"not null comment('用户昵称') VARCHAR(50)" json:"nickname"`
	Avatar        string    `xorm:"not null comment('用户头像') VARCHAR(1024)" json:"avatar"`
	Intro         string    `xorm:"comment('用户简介') VARCHAR(255)" json:"intro"`
	Website       string    `xorm:"comment('个人网站') VARCHAR(255)" json:"website"`
	IsSubscribe   int       `xorm:"comment('是否订阅') SMALLINT" json:"isSubscribe"`
	NotifyComment int       `xorm:"notify_comment not null default 1 comment('是否接收评论邮件通知') SMALLINT" json:"notifyComment"`
	IsDisable     int       `xorm:"not null comment('是否禁用') SMALLINT" json:"isDisable"`
	CreateTime    time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime    time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TUserRole struct {
	Id     int `xorm:"autoincr not null pk unique INTEGER"`
	UserId int `xorm:"comment('用户id') INTEGER"`
	RoleId int `xorm:"comment('角色id') INTEGER"`
}

type TWebsiteConfig struct {
	Id         int       `xorm:"autoincr not null pk unique INTEGER"`
	Config     string    `xorm:"comment('配置信息') VARCHAR"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME"`
}

// TNewsletterSubscriber is deliberately independent from TUserInfo. Public
// readers can subscribe without creating an account, while existing account
// subscriptions remain supported by the legacy user field.
type TNewsletterSubscriber struct {
	Id                    int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	Email                 string    `xorm:"not null unique VARCHAR(254)" json:"email"`
	Status                string    `xorm:"not null index VARCHAR(20)" json:"status"`
	ConfirmTokenHash      string    `xorm:"not null VARCHAR(64)"`
	ConfirmTokenExpiresAt time.Time `xorm:"not null DATETIME"`
	UnsubscribeTokenHash  string    `xorm:"not null VARCHAR(64)"`
	ConfirmedAt           time.Time `xorm:"DATETIME" json:"confirmedAt"`
	CreatedAt             time.Time `xorm:"created not null DATETIME" json:"createdAt"`
	UpdatedAt             time.Time `xorm:"updated DATETIME" json:"updatedAt"`
}

type TNewsletterDelivery struct {
	Id              int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	SubscriberId    int       `xorm:"not null index INTEGER" json:"subscriberId"`
	ArticleId       int       `xorm:"not null index INTEGER" json:"articleId"`
	Status          string    `xorm:"not null index VARCHAR(20)" json:"status"`
	Attempts        int       `xorm:"not null INTEGER" json:"attempts"`
	LastError       string    `xorm:"TEXT" json:"lastError"`
	SentAt          time.Time `xorm:"DATETIME" json:"sentAt"`
	CreatedAt       time.Time `xorm:"created not null DATETIME" json:"createdAt"`
	UpdatedAt       time.Time `xorm:"updated DATETIME" json:"updatedAt"`
	SubscriberEmail string    `json:"subscriberEmail"`
	ArticleTitle    string    `json:"articleTitle"`
}

// TGrowthEvent stores only an allow-listed event, an optional article id and
// a coarse path. It intentionally has no IP, user-agent, referrer or account
// identifier so the first-party analytics remain privacy-friendly.
type TGrowthEvent struct {
	Id        int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	EventName string    `xorm:"not null index VARCHAR(32)" json:"eventName"`
	ArticleId int       `xorm:"index INTEGER" json:"articleId"`
	Path      string    `xorm:"VARCHAR(255)" json:"path"`
	CreatedAt time.Time `xorm:"created not null index DATETIME" json:"createdAt"`
}

// TArticleReaction records one reader reaction per article and account. The
// unique constraint is what makes the toggle idempotent.
type TArticleReaction struct {
	Id         int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	ArticleId  int       `xorm:"article_id not null index INTEGER" json:"articleId"`
	UserInfoId int       `xorm:"user_info_id not null index INTEGER" json:"userInfoId"`
	Reaction   string    `xorm:"reaction not null VARCHAR(16)" json:"reaction"`
	CreateTime time.Time `xorm:"create_time created not null DATETIME" json:"createTime"`
}
