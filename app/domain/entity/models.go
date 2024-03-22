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
	Id             int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
	UserId         int       `xorm:"not null comment('作者') INTEGER" json:"userId"`
	CategoryId     int       `xorm:"comment('文章分类') INTEGER" json:"categoryId"`
	ArticleCover   string    `xorm:"comment('文章缩略图') VARCHAR(1024)" json:"articleCover"`
	ArticleTitle   string    `xorm:"not null comment('标题') VARCHAR(50)" json:"articleTitle"`
	ArticleContent string    `xorm:"not null comment('内容') TEXT" json:"articleContent"`
	IsTop          int       `xorm:"not null comment('是否置顶 0否 1是') SMALLINT" json:"isTop"`
	IsFeatured     int       `xorm:"not null comment('是否推荐 0否 1是') SMALLINT" json:"isFeatured"`
	IsDelete       int       `xorm:"not null comment('是否删除  0否 1是') SMALLINT" json:"isDelete"`
	Status         int       `xorm:"not null comment('状态值 1公开 2私密 3草稿') SMALLINT" json:"status"`
	Type           int       `xorm:"not null comment('文章类型 1原创 2转载 3翻译') SMALLINT" json:"type"`
	Password       string    `xorm:"comment('访问密码') VARCHAR(255)" json:"password"`
	OriginalUrl    string    `xorm:"comment('原文链接') VARCHAR(255)" json:"originalUrl"`
	CreateTime     time.Time `xorm:"created not null comment('发表时间') DATETIME" json:"createTime"`
	UpdateTime     time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TArticleTag struct {
	Id        int `xorm:"autoincr not null pk unique INTEGER"`
	ArticleId int `xorm:"not null comment('文章id') index INTEGER"`
	TagId     int `xorm:"not null comment('标签id') index INTEGER"`
}

type TCategory struct {
	Id           int       `xorm:"autoincr not null pk unique INTEGER" json:"id"`
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
	Id          int       `xorm:"autoincr not null pk unique INTEGER"`
	LinkName    string    `xorm:"not null comment('链接名') index VARCHAR(20)"`
	LinkAvatar  string    `xorm:"not null comment('链接头像') VARCHAR(255)"`
	LinkAddress string    `xorm:"not null comment('链接地址') VARCHAR(50)"`
	LinkIntro   string    `xorm:"not null comment('链接介绍') VARCHAR(100)"`
	CreateTime  time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime  time.Time `xorm:"updated comment('更新时间') DATETIME"`
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
	Id            int       `xorm:"autoincr not null pk comment('主键id') unique INTEGER"`
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
	CreateTime    time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime    time.Time `xorm:"updated comment('更新时间') DATETIME"`
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
	TagName    string    `xorm:"not null comment('标签名') VARCHAR(20)" json:"tagName"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TTalk struct {
	Id         int       `xorm:"autoincr not null pk comment('说说id') unique INTEGER" json:"id"`
	UserId     int       `xorm:"not null comment('用户id') INTEGER" json:"userId"`
	Content    string    `xorm:"not null comment('说说内容') VARCHAR(2000)" json:"content"`
	Images     string    `xorm:"comment('图片') VARCHAR(2500)" json:"images"`
	IsTop      int       `xorm:"not null comment('是否置顶') SMALLINT" json:"isTop"`
	Status     int       `xorm:"not null comment('状态 1.公开 2.私密') SMALLINT" json:"status"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
}

type TUniqueView struct {
	Id         int       `xorm:"autoincr not null pk unique INTEGER"`
	ViewsCount int       `xorm:"not null comment('访问量') INTEGER"`
	CreateTime time.Time `xorm:"created not null comment('创建时间') DATETIME"`
	UpdateTime time.Time `xorm:"updated comment('更新时间') DATETIME"`
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
	Id          int       `xorm:"autoincr not null pk comment('用户ID') unique INTEGER" json:"id"`
	Email       string    `xorm:"comment('邮箱号') VARCHAR(50)" json:"email"`
	Nickname    string    `xorm:"not null comment('用户昵称') VARCHAR(50)" json:"nickname"`
	Avatar      string    `xorm:"not null comment('用户头像') VARCHAR(1024)" json:"avatar"`
	Intro       string    `xorm:"comment('用户简介') VARCHAR(255)" json:"intro"`
	Website     string    `xorm:"comment('个人网站') VARCHAR(255)" json:"website"`
	IsSubscribe int       `xorm:"comment('是否订阅') SMALLINT" json:"isSubscribe"`
	IsDisable   int       `xorm:"not null comment('是否禁用') SMALLINT" json:"isDisable"`
	CreateTime  time.Time `xorm:"created not null comment('创建时间') DATETIME" json:"createTime"`
	UpdateTime  time.Time `xorm:"updated comment('更新时间') DATETIME" json:"updateTime"`
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
