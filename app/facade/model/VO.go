package model

import (
	"time"
)

type AboutVO struct {
	Content string `json:"content"`
}

type ArticlePasswordVO struct {
	ArticleId       int    `json:"articleId"`
	ArticlePassword string `json:"articlePassword"`
}

type ArticleTopFeaturedVO struct {
	Id         int `json:"id" form:"id"`
	IsTop      int `json:"isTop" form:"isTop"`
	IsFeatured int `json:"isFeatured" form:"isFeatured"`
}

type ArticleVO struct {
	Id             int      `json:"id" form:"id"`
	CategoryName   string   `json:"categoryName" form:"categoryName"`
	ArticleCover   string   `json:"articleCover" form:"articleCover"`
	ArticleTitle   string   `json:"articleTitle" form:"articleTitle"`
	ArticleContent string   `json:"articleContent" from:"articleContent"`
	TagNames       []string `json:"tagNames" form:"tagNames"`
	IsTop          int      `json:"isTop" from:"isTop"`
	IsFeatured     int      `json:"isFeatured" form:"isFeatured"`
	Status         int      `json:"status" from:"status"`
	Type           int      `json:"type" form:"type"`
	Password       string   `json:"password" form:"password"`
	OriginalUrl    string   `json:"originalUrl" from:"originalUrl"`
}

type CategoryVO struct {
	Id           int    `json:"id"`
	CategoryName string `json:"categoryName"`
}

type CommentVO struct {
	TopicId        string `json:"topicId" form:"topicId"`
	CommentContent string `json:"commentContent" form:"commentContent"`
	ReplyUserId    int    `json:"replyUserId" form:"replyUserId"`
	ParentId       int    `json:"parentId" form:"parentId"`
	Type           int    `json:"type" form:"type"`
}

type ConditionVO struct {
	Current    int       `json:"current" form:"current"`
	Size       int       `json:"size" form:"size"`
	Keywords   string    `json:"keywords" form:"keywords"`
	CategoryId int       `json:"categoryId" form:"categoryId"`
	TagId      int       `json:"tagId" form:"tagId"`
	AlbumId    int       `json:"albumId" form:"albumId"`
	LonginType int       `json:"longinType" form:"longinType"`
	Type       int       `json:"type" form:"type"`
	Status     int       `json:"status" form:"status"`
	StartTime  time.Time `json:"startTime" form:"startTime"`
	EndTime    time.Time `json:"endTime" form:"endTime"`
	IsDelete   int       `json:"isDelete" form:"isDelete"`
	IsReview   int       `json:"isReview" form:"isReview"`
	IsTop      int       `json:"isTop" form:"isTop"`
	IsFeatured int       `json:"isFeatured" form:"isFeatured"`
}

type DeleteVO struct {
	Ids      []int `json:"ids" form:"ids"`
	IsDelete int   `json:"isDelete" form:"isDelete"`
}

type EmailVO struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type FriendLinkVO struct {
	Id          int    `json:"id"`
	LinkName    string `json:"linkName"`
	LinkAvatar  string `json:"linkAvatar"`
	LinkAddress string `json:"linkAddress"`
	LinkIntro   string `json:"linkIntro"`
}

type IsHiddenVO struct {
	Id       int `json:"id"`
	IsHidden int `json:"isHidden"`
}

type JobLogSearchVO struct {
	JobId     int    `json:"jobId"`
	JobName   string `json:"jobName"`
	JobGroup  string `json:"jobGroup"`
	Status    any    `json:"status"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type JobRunVO struct {
	Id       int    `json:"id"`
	JobGroup string `json:"jobGroup"`
}

type JobSearchVO struct {
	JobName  string `json:"jobName"`
	JobGroup string `json:"jobGroup"`
	Status   int    `json:"status"`
}

type JobStatusVO struct {
	Id     int `json:"id"`
	Status int `json:"status"`
}

type JobVO struct {
	Id             int    `json:"id"`
	JobName        string `json:"jobName"`
	JobGroup       string `json:"jobGroup"`
	InvokeTarget   string `json:"invokeTarget"`
	CronExpression string `json:"cronExpression"`
	MisfirePolicy  int    `json:"misfirePolicy"`
	Concurrent     int    `json:"concurrent"`
	Status         int    `json:"status"`
	Remark         string `json:"remark"`
}

type MenuVO struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	OrderNum  int    `json:"orderNum"`
	ParentId  int    `json:"parentId"`
	IsHidden  int    `json:"IsHidden"`
}

type PasswordVO struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type PhotoVO struct {
	AlbumId   string   `json:"albumId"`
	PhotoUrls []string `json:"photoUrls"`
	PhotoIds  []int    `json:"photoIds"`
}

type PhotoVO1 struct {
	AlbumId   int      `json:"albumId"`
	PhotoUrls []string `json:"photoUrls"`
	PhotoIds  []int    `json:"photoIds"`
}

type PhotoAlbumVO struct {
	Id         int    `json:"id"`
	AlbumName  string `json:"albumName"`
	AlbumDesc  string `json:"albumDesc"`
	AlbumCover string `json:"albumCover"`
	Status     int    `json:"status"`
}

type PhotoInfoVO struct {
	Id        int    `json:"id"`
	PhotoName string `json:"photoName"`
	PhotoDesc string `json:"photoDesc"`
}

type QQLoginVO struct {
	OpenId      string `json:"openId"`
	AccessToken string `json:"accessToken"`
}

type ResourceVO struct {
	Id            int    `json:"id"`
	ResourceName  string `json:"resourceName"`
	Url           string `json:"url"`
	RequestMethod string `json:"requestMethod"`
	ParentId      int    `json:"parentId"`
	IsAnonymous   int    `json:"isAnonymous"`
}

type ReviewVO struct {
	Ids      []int `json:"ids"`
	IsReview int   `json:"isReview"`
}

type RoleVO struct {
	Id          int    `json:"id"`
	RoleName    string `json:"roleName"`
	ResourceIds []int  `json:"resourceIds"`
	MenuIds     []int  `json:"menuIds"`
}

type SubscribeVO struct {
	UserId      int `json:"userId"`
	IsSubscribe int `json:"isSubscribe"`
}

type TagVO struct {
	Id      int    `json:"id"`
	TagName string `json:"tagName"`
}

type TalkVO struct {
	Id      int    `json:"id"`
	Content string `json:"content"`
	Images  string `json:"images"`
	IsTop   int    `json:"isTop"`
	Status  int    `json:"status"`
}

type UserDisableVO struct {
	Id        int `json:"id"`
	IsDisable int `json:"isDisable"`
}

type UserInfoVO struct {
	Nickname string `json:"nickname"`
	Intro    string `json:"intro"`
	Website  string `json:"website"`
}

type UserRoleVO struct {
	UserInfoId int    `json:"userInfoId"`
	NickName   string `json:"nickname"`
	RoleIds    []int  `json:"roleIds"`
}

type UserVO struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
	Code     string `json:"code" form:"code"`
}

type WebsiteConfigVO struct {
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
