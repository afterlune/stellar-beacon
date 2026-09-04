package port

import (
	"context"
	"time"
)

// SiteInfoRepository owns the small read/write model used by the public site
// and the administration dashboard.  The cache remains an application
// concern; this port only describes the database fallback and persistence.
type SiteInfoRepository interface {
	CountArticles(ctx context.Context) (int64, error)
	CountCategories(ctx context.Context) (int64, error)
	CountTags(ctx context.Context) (int64, error)
	CountTalks(ctx context.Context) (int64, error)
	CountRecentContent(ctx context.Context, since time.Time) (int64, error)
	CountComments(ctx context.Context, commentType int) (int64, error)
	CountUsers(ctx context.Context) (int64, error)
	ListUniqueViews(ctx context.Context, startTime, endTime string) ([]UniqueView, error)
	ListArticleRank(ctx context.Context, ids []int) ([]ArticleRank, error)
	GetWebsiteConfig(ctx context.Context) (string, error)
	UpdateWebsiteConfig(ctx context.Context, config string) error
	GetAbout(ctx context.Context, id int) (string, error)
	UpdateAbout(ctx context.Context, id int, content string) error
}

type UniqueView struct {
	Day        string `json:"day"`
	ViewsCount int    `json:"viewsCount"`
}

type ArticleRank struct {
	Id           int    `json:"id"`
	ArticleTitle string `json:"articleTitle"`
	ViewsCount   int    `json:"viewsCount"`
}

type FriendLinkRepository interface {
	ListPublic(ctx context.Context) ([]TFriendLink, error)
	ListAdmin(ctx context.Context, current, size int, keywords string) ([]TFriendLink, int64, error)
	SaveOrUpdate(ctx context.Context, link TFriendLink) error
	Delete(ctx context.Context, ids []int) error
}

type JobFilter struct {
	JobName  string
	JobGroup string
	Status   int
}

type JobRepository interface {
	Get(ctx context.Context, id int) (TJob, error)
	List(ctx context.Context, current, size int, filter JobFilter) ([]TJob, int, error)
	ListGroups(ctx context.Context) ([]string, error)
	Save(ctx context.Context, job TJob) error
	Update(ctx context.Context, job TJob) error
	Delete(ctx context.Context, ids []int) error
	UpdateStatus(ctx context.Context, id, status int) error
}

type JobLogFilter struct {
	JobId     int
	JobName   string
	JobGroup  string
	Status    *int
	StartTime string
	EndTime   string
}

type JobLogRepository interface {
	List(ctx context.Context, current, size int, filter JobLogFilter) ([]TJobLog, int64, error)
	Delete(ctx context.Context, ids []int) error
	Clean(ctx context.Context) error
	ListGroups(ctx context.Context) (string, error)
}

type ErrorLogRepository interface {
	List(ctx context.Context, current, size int, keywords string) ([]TExceptionLog, int64, error)
	Delete(ctx context.Context, ids []int) error
}

type OperationLogRepository interface {
	List(ctx context.Context, current, size int, keywords string) ([]TOperationLog, int64, error)
	Delete(ctx context.Context, ids []int) error
}

type MenuRepository interface {
	List(ctx context.Context, keywords string) ([]TMenu, error)
	ListOptions(ctx context.Context) ([]TMenu, error)
	ListByUserInfoID(ctx context.Context, userInfoID int) ([]TMenu, error)
	SaveOrUpdate(ctx context.Context, menu TMenu) error
	UpdateHidden(ctx context.Context, id, hidden int) error
	Delete(ctx context.Context, id int) error
}

type ResourceRepository interface {
	List(ctx context.Context, keywords string) ([]TResource, error)
	ListOptions(ctx context.Context) ([]TResource, error)
	SaveOrUpdate(ctx context.Context, resource TResource) error
	Delete(ctx context.Context, id int) error
}

type RoleView struct {
	Id          int       `json:"id"`
	RoleName    string    `json:"roleName"`
	CreateTime  time.Time `json:"createTime"`
	IsDisable   int       `json:"isDisable"`
	ResourceIds []int     `json:"resourceIds"`
	MenuIds     []int     `json:"menuIds"`
}

type ResourceRoleView struct {
	Id            int      `json:"id"`
	Url           string   `json:"url"`
	RequestMethod string   `json:"requestMethod"`
	RoleList      []string `json:"roleList"`
}

type RoleRepository interface {
	ListUserRoles(ctx context.Context) ([]TRole, error)
	Count(ctx context.Context, keywords string) (int64, error)
	List(ctx context.Context, current, size int, keywords string) ([]RoleView, error)
	FindByName(ctx context.Context, name string) (TRole, error)
	SaveOrUpdate(ctx context.Context, role TRole, resourceIDs, menuIDs []int) error
	Delete(ctx context.Context, ids []int) error
	ListResourceRoles(ctx context.Context) ([]ResourceRoleView, error)
	ListRolesByUserInfoID(ctx context.Context, userInfoID int) ([]string, error)
}

type PhotoAlbumAdmin struct {
	Id         int    `json:"id"`
	AlbumName  string `json:"albumName"`
	AlbumDesc  string `json:"albumDesc"`
	AlbumCover string `json:"albumCover"`
	PhotoCount int    `json:"photoCount"`
	Status     int    `json:"status"`
}

type PhotoAlbumRepository interface {
	ListPublic(ctx context.Context) ([]TPhotoAlbum, error)
	FindByName(ctx context.Context, name string) (TPhotoAlbum, error)
	ListAdmin(ctx context.Context, current, size int, keywords string) ([]PhotoAlbumAdmin, int64, error)
	ListOptions(ctx context.Context) ([]TPhotoAlbum, error)
	Get(ctx context.Context, id int) (TPhotoAlbum, error)
	SaveOrUpdate(ctx context.Context, album TPhotoAlbum) error
	Delete(ctx context.Context, id int) error
}

type PhotoRepository interface {
	List(ctx context.Context, current, size, albumID, isDelete int) ([]TPhoto, int64, error)
	Update(ctx context.Context, photo TPhoto) error
	InsertMany(ctx context.Context, photos []TPhoto) error
	UpdateAlbum(ctx context.Context, ids []int, albumID int) error
	UpdateDelete(ctx context.Context, ids []int, isDelete int) error
	Delete(ctx context.Context, ids []int) error
	ListPublicByAlbum(ctx context.Context, albumID, current, size int) ([]TPhoto, error)
}

type UserInfoRepository interface {
	UpdateProfile(ctx context.Context, id int, nickname, intro, website string) error
	UpdateAvatar(ctx context.Context, id int, avatar string) error
	GetByID(ctx context.Context, id int) (TUserInfo, error)
	UpdateEmail(ctx context.Context, id int, email string) error
	UpdateSubscribe(ctx context.Context, id, subscribe int) error
	UpdateRole(ctx context.Context, userInfoID int, nickname string, roleIDs []int) error
	UpdateDisable(ctx context.Context, id, disabled int) error
	FindAuthByUserInfoID(ctx context.Context, id int) (TUserAuth, error)
}
