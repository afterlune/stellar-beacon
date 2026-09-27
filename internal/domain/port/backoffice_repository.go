package port

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
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

// Friend link review states: submissions are pending until an administrator
// approves or rejects them, and only approved links are public.
const (
	FriendLinkStatusPending  = 0
	FriendLinkStatusApproved = 1
	FriendLinkStatusRejected = 2
)

type FriendLinkRepository interface {
	// ListPublic returns approved links only.
	ListPublic(ctx context.Context) ([]entity.TFriendLink, error)
	ListAdmin(ctx context.Context, current, size int, keywords string) ([]entity.TFriendLink, int64, error)
	SaveOrUpdate(ctx context.Context, link entity.TFriendLink) error
	Delete(ctx context.Context, ids []int) error
	// CreateApplication stores a reader submission as pending.
	CreateApplication(ctx context.Context, link entity.TFriendLink) (int, error)
	// Review approves or rejects submissions.
	Review(ctx context.Context, ids []int, status int) error
	// AddressExists reports whether a link with the same address is stored.
	AddressExists(ctx context.Context, address string) (bool, error)
}

type JobFilter struct {
	JobName  string
	JobGroup string
	Status   *int
}

type JobRepository interface {
	Get(ctx context.Context, id int) (entity.TJob, error)
	List(ctx context.Context, current, size int, filter JobFilter) ([]entity.TJob, int, error)
	ListEnabled(ctx context.Context) ([]entity.TJob, error)
	ListGroups(ctx context.Context) ([]string, error)
	SaveOrUpdate(ctx context.Context, job entity.TJob) error
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
	List(ctx context.Context, current, size int, filter JobLogFilter) ([]entity.TJobLog, int64, error)
	Create(ctx context.Context, log entity.TJobLog) error
	Delete(ctx context.Context, ids []int) error
	Clean(ctx context.Context) error
	CleanBefore(ctx context.Context, before time.Time) error
	ListGroups(ctx context.Context) (string, error)
}

type ErrorLogRepository interface {
	List(ctx context.Context, current, size int, keywords string) ([]entity.TExceptionLog, int64, error)
	Delete(ctx context.Context, ids []int) error
}

type OperationLogRepository interface {
	List(ctx context.Context, current, size int, keywords string) ([]entity.TOperationLog, int64, error)
	Delete(ctx context.Context, ids []int) error
}

type MenuRepository interface {
	List(ctx context.Context, keywords string) ([]entity.TMenu, error)
	ListOptions(ctx context.Context) ([]entity.TMenu, error)
	ListByUserInfoID(ctx context.Context, userInfoID int) ([]entity.TMenu, error)
	SaveOrUpdate(ctx context.Context, menu entity.TMenu) error
	UpdateHidden(ctx context.Context, id, hidden int) error
	Delete(ctx context.Context, id int) error
}

type ResourceRepository interface {
	List(ctx context.Context, keywords string) ([]entity.TResource, error)
	ListOptions(ctx context.Context) ([]entity.TResource, error)
	SaveOrUpdate(ctx context.Context, resource entity.TResource) error
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
	ListUserRoles(ctx context.Context) ([]entity.TRole, error)
	Count(ctx context.Context, keywords string) (int64, error)
	List(ctx context.Context, current, size int, keywords string) ([]RoleView, error)
	FindByName(ctx context.Context, name string) (entity.TRole, error)
	SaveOrUpdate(ctx context.Context, role entity.TRole, resourceIDs, menuIDs []int) error
	Delete(ctx context.Context, ids []int) error
	ListResourceRoles(ctx context.Context) ([]ResourceRoleView, error)
	ListRolesByUserInfoID(ctx context.Context, userInfoID int) ([]string, error)
	HasUserResourcePermission(ctx context.Context, userInfoID int, path, method string) (bool, error)
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
	ListPublic(ctx context.Context) ([]entity.TPhotoAlbum, error)
	ListPublicByHandle(ctx context.Context, handle string) ([]entity.TPhotoAlbum, error)
	GetPublicByHandle(ctx context.Context, handle string, albumID int) (entity.TPhotoAlbum, error)
	ListOwned(ctx context.Context, userID int) ([]entity.TPhotoAlbum, error)
	GetOwned(ctx context.Context, id, userID int) (entity.TPhotoAlbum, error)
	SaveOwned(ctx context.Context, album entity.TPhotoAlbum, userID int) error
	DeleteOwned(ctx context.Context, id, userID int) error
	FindByName(ctx context.Context, name string) (entity.TPhotoAlbum, error)
	ListAdmin(ctx context.Context, current, size int, keywords string) ([]PhotoAlbumAdmin, int64, error)
	ListOptions(ctx context.Context) ([]entity.TPhotoAlbum, error)
	Get(ctx context.Context, id int) (entity.TPhotoAlbum, error)
	SaveOrUpdate(ctx context.Context, album entity.TPhotoAlbum) error
	Delete(ctx context.Context, id int) error
}

type PhotoRepository interface {
	List(ctx context.Context, current, size, albumID, isDelete int, keywords string) ([]entity.TPhoto, int64, error)
	ListOwnedByAlbum(ctx context.Context, userID, albumID int) ([]entity.TPhoto, error)
	InsertOwned(ctx context.Context, userID, albumID int, photos []entity.TPhoto) error
	DeleteOwned(ctx context.Context, userID int, photoIDs []int) error
	Update(ctx context.Context, photo entity.TPhoto) error
	InsertMany(ctx context.Context, photos []entity.TPhoto) error
	UpdateAlbum(ctx context.Context, ids []int, albumID int) error
	UpdateDelete(ctx context.Context, ids []int, isDelete int) error
	Delete(ctx context.Context, ids []int) error
	ListPublicByAlbum(ctx context.Context, albumID, current, size int) ([]entity.TPhoto, error)
}

type UserInfoRepository interface {
	UpdateProfile(ctx context.Context, id int, nickname, intro, website string) error
	UpdateAvatar(ctx context.Context, id int, avatar string) error
	GetByID(ctx context.Context, id int) (entity.TUserInfo, error)
	UpdateEmail(ctx context.Context, id int, email string) error
	UpdateSubscribe(ctx context.Context, id, subscribe int) error
	UpdateNotifyComment(ctx context.Context, id, notify int) error
	UpdateNotifyInteraction(ctx context.Context, id, notify int) error
	UpdateNotifyTopic(ctx context.Context, id, notify int) error
	UpdateNotifyCollection(ctx context.Context, id, notify int) error
	UpdateNotifyStudioActivation(ctx context.Context, id, notify int) error
	UpdateRole(ctx context.Context, userInfoID int, nickname string, roleIDs []int) error
	UpdateDisable(ctx context.Context, id, disabled int) error
	IsEnabled(ctx context.Context, id int) (bool, error)
	FindAuthByUserInfoID(ctx context.Context, id int) (entity.TUserAuth, error)
}
